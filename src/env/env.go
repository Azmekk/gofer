package env

import (
	"bufio"
	"os"
	"strings"
)

func LoadEnvFile(path string) (map[string]string, error) {
	vars := make(map[string]string)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vars, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vars[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return vars, scanner.Err()
}

func BuildEnv(envFileVars map[string]string) []string {
	envMap := make(map[string]string)

	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			envMap[key] = value
		}
	}

	for k, v := range envFileVars {
		envMap[k] = v
	}

	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}
