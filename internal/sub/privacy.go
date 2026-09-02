package sub

import (
	"html/template"
	"strings"
)

// The subscription page is a public surface. User.Name and the user's device roster
// are operator-only data, so neither may be part of the template we execute. Keep
// exact upstream markers on purpose: if upstream changes either block, fail loudly
// at startup instead of silently re-exposing private data after a sync.
func init() {
	const greeting = "      <h1>{{.L.Greeting}}{{if .Name}}, {{.Name}}{{end}} 👋</h1>\n"
	if !strings.Contains(pageHTML, greeting) {
		panic("subscription privacy: greeting marker not found")
	}
	pageHTML = strings.Replace(pageHTML, greeting, "", 1)

	const devices = `      {{if .Devices.Show}}
      <div class="card" id="devices-card">
        <div class="lbl" style="margin-bottom: 8px">
          {{.L.Devices}} <span class="muted">{{.Devices.CountText}}</span>
        </div>
        {{range .Devices.List}}
        <div class="dev" data-hwid="{{.HWID}}">
          <div class="dev-info">
            <div class="dev-name">{{.Title}}</div>
            <div class="muted dev-sub">
              {{if .Sub}}{{.Sub}} · {{end}}{{.LastSeen}}
            </div>
            <div class="muted dev-id">{{.HWID}}</div>
          </div>
          <button class="btn alt dev-del" onclick="unbindDevice(this)">
            {{$.L.DeviceRemove}}
          </button>
        </div>
        {{else}}
        <p class="muted" style="margin: 0">{{.L.DevicesEmpty}}</p>
        {{end}}
        <p class="muted" style="margin: 10px 0 0">{{.L.DevicesHint}}</p>
      </div>
      {{end}}
`
	if !strings.Contains(pageHTML, devices) {
		panic("subscription privacy: devices marker not found")
	}
	pageHTML = strings.Replace(pageHTML, devices, "", 1)

	pageTmpl = template.Must(template.New("sub").Parse(pageHTML))
}
