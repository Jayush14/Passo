package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed *.tmpl
var tmplFS embed.FS

// RenderTemplate renders a named template from the embedded templates.
func RenderTemplate(name string, data interface{}) (string, error) {
	b, err := tmplFS.ReadFile(name + ".tmpl")
	if err != nil {
		return "", fmt.Errorf("template %s not found: %w", name, err)
	}
	t, err := template.New(name).Parse(string(b))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return buf.String(), nil
}
