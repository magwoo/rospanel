package sub

import (
	"html/template"
	"strings"
)

// The subscription page is a public surface. User.Name is an operator-only label,
// so the stock personalized greeting must never be part of the template we execute.
// Keep the exact marker here on purpose: if upstream changes that markup, fail loudly
// at startup instead of silently re-exposing names after a rebase.
func init() {
	const greeting = "      <h1>{{.L.Greeting}}{{if .Name}}, {{.Name}}{{end}} 👋</h1>\n"
	if !strings.Contains(pageHTML, greeting) {
		panic("subscription privacy: greeting marker not found")
	}
	pageHTML = strings.Replace(pageHTML, greeting, "", 1)
	pageTmpl = template.Must(template.New("sub").Parse(pageHTML))
}
