package sub

import (
	"strings"
	"testing"

	"github.com/AppsGanin/rospanel/internal/model"
)

func TestSubTitleNeverExposesInternalUserName(t *testing.T) {
	u := model.User{Name: "PRIVATE-INTERNAL-NAME"}
	set := &model.Settings{SubTitle: "Travel VPN", SubNameInTitle: true}

	got := SubTitle(u, set)
	if got != "Travel VPN" {
		t.Fatalf("SubTitle() = %q, want operator title only", got)
	}
	if strings.Contains(got, u.Name) {
		t.Fatalf("SubTitle exposed internal user name: %q", got)
	}
}

func TestSubscriptionPageTemplateHasNoPersonalGreeting(t *testing.T) {
	const marker = "{{.L.Greeting}}{{if .Name}}, {{.Name}}{{end}}"
	if strings.Contains(pageHTML, marker) {
		t.Fatal("public subscription page still contains the personalized greeting")
	}
}
