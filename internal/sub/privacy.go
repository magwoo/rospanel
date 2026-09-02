package sub

import (
	"html/template"
	"strings"
)

// The subscription page is a public surface. User.Name and the user's device roster
// are operator-only data, so neither may be part of the template we execute. Keep
// explicit upstream boundaries on purpose: if upstream changes either surface, fail
// loudly at startup instead of silently re-exposing private data after a sync.
func init() {
	const greeting = "      <h1>{{.L.Greeting}}{{if .Name}}, {{.Name}}{{end}} 👋</h1>\n"
	if !strings.Contains(pageHTML, greeting) {
		panic("subscription privacy: greeting marker not found")
	}
	pageHTML = strings.Replace(pageHTML, greeting, "", 1)

	const deviceStart = "      {{if .Devices.Show}}\n"
	const configsStart = "      {{if .ShowConfigs}}\n"
	start := strings.Index(pageHTML, deviceStart)
	if start < 0 {
		panic("subscription privacy: devices start marker not found")
	}
	rest := pageHTML[start:]
	end := strings.Index(rest, configsStart)
	if end < 0 {
		panic("subscription privacy: devices end marker not found")
	}
	pageHTML = pageHTML[:start] + rest[end:]

	pageTmpl = template.Must(template.New("sub").Parse(pageHTML))
}
