package executor

import (
	"bytes"
	"fmt"
	"text/template"
)

func ResolveTemplate(cmdStr string, params map[string]string) (string, error) {
	tmpl, err := template.New("cmd").Option("missingkey=error").Parse(cmdStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %q: %w", cmdStr, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("failed to resolve template %q: %w", cmdStr, err)
	}

	return buf.String(), nil
}
