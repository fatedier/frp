package cobra

import (
	"testing"
	"text/template"
)

func TestAddTemplateFunctions(t *testing.T) {
	AddTemplateFunc("t", func() bool { return true })
	AddTemplateFuncs(template.FuncMap{
		"f": func() bool { return false },
		"h": func() string { return "Hello," },
		"w": func() string { return "world." }})

	c := &Command{}
	c.SetUsageTemplate(`{{if t}}{{h}}{{end}}{{if f}}{{h}}{{end}} {{w}}`)

	const expected = "Hello, world."
	if got := c.UsageString(); got != expected {
		t.Errorf("Expected UsageString: %v\nGot: %v", expected, got)
	}
}
