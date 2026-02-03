package schema

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed gofer_schema.json
var SchemaJSON []byte

func Validate(data []byte) []error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return []error{fmt.Errorf("invalid JSON: %w", err)}
	}

	var errs []error

	tasksRaw, ok := raw["tasks"]
	if !ok {
		errs = append(errs, fmt.Errorf("missing required field: tasks"))
		return errs
	}

	tasks, ok := tasksRaw.(map[string]interface{})
	if !ok {
		errs = append(errs, fmt.Errorf("tasks must be an object"))
		return errs
	}

	// Check for duplicate task keys in raw JSON
	errs = append(errs, checkDuplicateTaskKeys(data)...)

	for tName, tRaw := range tasks {
		errs = append(errs, validateTask(tName, tRaw)...)
	}

	return errs
}

func checkDuplicateTaskKeys(data []byte) []error {
	var errs []error
	dec := json.NewDecoder(bytes.NewReader(data))

	// Find the "tasks" key at the top level
	depth := 0
	foundTasks := false
	for {
		tok, err := dec.Token()
		if err != nil {
			return errs
		}

		if delim, ok := tok.(json.Delim); ok {
			switch delim {
			case '{':
				depth++
			case '}':
				depth--
			case '[':
				depth++
			case ']':
				depth--
			}
			continue
		}

		if depth == 1 {
			if key, ok := tok.(string); ok && key == "tasks" {
				foundTasks = true
				// Next token should be '{' opening the tasks object
				tok, err = dec.Token()
				if err != nil {
					return errs
				}
				if delim, ok := tok.(json.Delim); !ok || delim != '{' {
					return errs
				}
				break
			} else {
				// Skip the value for this non-tasks key
				skipValue(dec)
			}
		}
	}

	if !foundTasks {
		return errs
	}

	// Now read keys inside the tasks object
	seen := make(map[string]bool)
	taskDepth := 1
	readingKey := true
	for taskDepth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return errs
		}

		if delim, ok := tok.(json.Delim); ok {
			switch delim {
			case '{', '[':
				taskDepth++
				readingKey = false
			case '}', ']':
				taskDepth--
				if taskDepth == 1 {
					readingKey = true
				}
			}
			continue
		}

		if taskDepth == 1 && readingKey {
			if key, ok := tok.(string); ok {
				if seen[key] {
					errs = append(errs, fmt.Errorf("duplicate task name: %q", key))
				}
				seen[key] = true
				readingKey = false
			}
		} else if taskDepth == 1 && !readingKey {
			// This is a primitive value for a task key (shouldn't happen, tasks are objects)
			readingKey = true
		}
	}

	return errs
}

func skipValue(dec *json.Decoder) {
	tok, err := dec.Token()
	if err != nil {
		return
	}
	if delim, ok := tok.(json.Delim); ok {
		if delim == '{' || delim == '[' {
			depth := 1
			for depth > 0 {
				t, err := dec.Token()
				if err != nil {
					return
				}
				if d, ok := t.(json.Delim); ok {
					switch d {
					case '{', '[':
						depth++
					case '}', ']':
						depth--
					}
				}
			}
		}
	}
}

func validateTask(path string, raw interface{}) []error {
	var errs []error

	task, ok := raw.(map[string]interface{})
	if !ok {
		return []error{fmt.Errorf("task %q must be an object", path)}
	}

	if _, ok := task["desc"]; !ok {
		errs = append(errs, fmt.Errorf("task %q: missing required field: desc", path))
	}

	stepsRaw, ok := task["steps"]
	if !ok {
		errs = append(errs, fmt.Errorf("task %q: missing required field: steps", path))
		return errs
	}

	steps, ok := stepsRaw.([]interface{})
	if !ok {
		errs = append(errs, fmt.Errorf("task %q: steps must be an array", path))
		return errs
	}

	for i, s := range steps {
		stepPath := fmt.Sprintf("%s.steps[%d]", path, i)
		errs = append(errs, validateStep(stepPath, s)...)
	}

	if paramsRaw, ok := task["params"]; ok {
		params, ok := paramsRaw.([]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("task %q: params must be an array", path))
		} else {
			for i, p := range params {
				paramPath := fmt.Sprintf("%s.params[%d]", path, i)
				errs = append(errs, validateParam(paramPath, p)...)
			}
		}
	}

	if groupRaw, ok := task["group"]; ok {
		if _, ok := groupRaw.(string); !ok {
			errs = append(errs, fmt.Errorf("task %q: group must be a string", path))
		}
	}

	return errs
}

func validateParam(path string, raw interface{}) []error {
	param, ok := raw.(map[string]interface{})
	if !ok {
		return []error{fmt.Errorf("param %q must be an object", path)}
	}
	if _, ok := param["name"]; !ok {
		return []error{fmt.Errorf("param %q: missing required field: name", path)}
	}
	return nil
}

func validateStep(path string, raw interface{}) []error {
	var errs []error

	step, ok := raw.(map[string]interface{})
	if !ok {
		return []error{fmt.Errorf("step %q must be an object", path)}
	}

	_, hasCmd := step["cmd"]
	_, hasRef := step["ref"]
	concurrentRaw, hasConcurrent := step["concurrent"]

	count := 0
	if hasCmd {
		count++
	}
	if hasRef {
		count++
	}
	if hasConcurrent {
		count++
	}

	if count == 0 {
		errs = append(errs, fmt.Errorf("step %q: must have exactly one of cmd, ref, or concurrent", path))
	} else if count > 1 {
		errs = append(errs, fmt.Errorf("step %q: must have exactly one of cmd, ref, or concurrent (found %d)", path, count))
	}

	if hasConcurrent {
		concurrent, ok := concurrentRaw.([]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("step %q: concurrent must be an array", path))
		} else {
			for i, s := range concurrent {
				subPath := fmt.Sprintf("%s.concurrent[%d]", path, i)
				errs = append(errs, validateStep(subPath, s)...)
			}
		}
	}

	if osVal, ok := step["os"]; ok {
		if osStr, ok := osVal.(string); ok {
			validOS := map[string]bool{"*": true, "": true, "linux": true, "darwin": true, "windows": true}
			if !validOS[osStr] {
				errs = append(errs, fmt.Errorf("step %q: invalid os value %q (must be linux, darwin, windows, or *)", path, osStr))
			}
		}
	}

	return errs
}
