package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AppsGanin/rospanel/internal/core"
	"github.com/AppsGanin/rospanel/internal/model"
	"github.com/AppsGanin/rospanel/internal/store"
)

// hwidUser creates a user with the given device cap and turns device binding on.
func hwidUser(t *testing.T, mgr *core.Manager, st *store.Store, capacity int, require bool) model.User {
	t.Helper()
	u, err := mgr.CreateUser(t.Context(), "hwid-user", 0, 0)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := mgr.SetUserLimits(t.Context(), u.ID, 0, 0, capacity); err != nil {
		t.Fatalf("set limits: %v", err)
	}
	set, err := st.GetSettings()
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	set.HWIDEnabled, set.HWIDRequire = true, require
	if err := st.SetHWIDSettings(set); err != nil {
		t.Fatalf("save hwid settings: %v", err)
	}
	fresh, err := st.GetUser(u.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	return *fresh
}

// fetchSub asks for the machine payload the way a client does, with the device
// headers Happ and v2RayTun send.
func fetchSub(h http.Handler, token, hwid string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/sub/"+token, nil)
	req.Header.Set("User-Agent", "Happ/1.0")
	if hwid != "" {
		req.Header.Set(model.HeaderHWID, hwid)
		req.Header.Set(model.HeaderDeviceOS, "android")
		req.Header.Set(model.HeaderDeviceModel, "Pixel 9")
	}
	req.RemoteAddr = testClientIP + ":40000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSubscriptionBindsDevicesUpToTheCap(t *testing.T) {
	h, mgr, st := nodeAPITestServer(t)
	u := hwidUser(t, mgr, st, 2, false)

	for _, hwid := range []string{"dev-a", "dev-b"} {
		if rec := fetchSub(h, u.SubToken, hwid); rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, want 200", hwid, rec.Code)
		}
	}
	if rec := fetchSub(h, u.SubToken, "dev-a"); rec.Code != http.StatusOK {
		t.Errorf("bound device refused on refetch: status %d", rec.Code)
	}

	rec := fetchSub(h, u.SubToken, "dev-c")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("third device: status %d, want 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "2") {
		t.Errorf("refusal body doesn't say how many devices are bound: %q", rec.Body.String())
	}
	devices, err := st.ListDevices(u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("%d devices bound, want 2", len(devices))
	}
	if devices[0].Model == "" || devices[0].OS == "" {
		t.Errorf("device headers not recorded: %+v", devices[0])
	}
}

func TestSubscriptionRefusesClientsWithoutHWID(t *testing.T) {
	h, mgr, st := nodeAPITestServer(t)
	u := hwidUser(t, mgr, st, 1, true)

	if rec := fetchSub(h, u.SubToken, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("client with no id: status %d, want 403", rec.Code)
	}

	set, _ := st.GetSettings()
	set.HWIDRequire = false
	if err := st.SetHWIDSettings(set); err != nil {
		t.Fatalf("save: %v", err)
	}
	if rec := fetchSub(h, u.SubToken, ""); rec.Code != http.StatusOK {
		t.Errorf("with the requirement off: status %d, want 200", rec.Code)
	}
}

func TestSubscriptionIgnoresDevicesWhenDisabled(t *testing.T) {
	h, mgr, st := nodeAPITestServer(t)
	u := hwidUser(t, mgr, st, 1, false)
	set, _ := st.GetSettings()
	set.HWIDEnabled = false
	if err := st.SetHWIDSettings(set); err != nil {
		t.Fatalf("save: %v", err)
	}

	for _, hwid := range []string{"dev-a", "dev-b", "dev-c"} {
		if rec := fetchSub(h, u.SubToken, hwid); rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, want 200", hwid, rec.Code)
		}
	}
	if n, _ := st.CountDevices(u.ID); n != 0 {
		t.Errorf("%d devices bound while the feature is off", n)
	}
}

// Device binding is operator-managed. The public subscription page exposes neither
// the roster nor its count, and the former self-service unbind path cannot mutate it.
func TestSubscriptionPageHidesDevicesAndCannotUnbind(t *testing.T) {
	h, mgr, st := nodeAPITestServer(t)
	u := hwidUser(t, mgr, st, 2, false)
	if rec := fetchSub(h, u.SubToken, "dev-a"); rec.Code != http.StatusOK {
		t.Fatalf("bind: status %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/sub/"+u.SubToken, nil)
	req.Header.Set("Accept", "text/html")
	req.RemoteAddr = testClientIP + ":40000"
	page := httptest.NewRecorder()
	h.ServeHTTP(page, req)
	if page.Code != http.StatusOK {
		t.Fatalf("page: status %d", page.Code)
	}
	for _, leaked := range []string{"Pixel 9", `data-hwid="dev-a"`, `id="devices-card"`} {
		if strings.Contains(page.Body.String(), leaked) {
			t.Errorf("subscription page exposes device data %q", leaked)
		}
	}

	unbind := httptest.NewRequest(http.MethodPost, "/sub/"+u.SubToken+"/devices/unbind", strings.NewReader(`{"hwid":"dev-a"}`))
	unbind.Header.Set("Content-Type", "application/json")
	unbind.RemoteAddr = testClientIP + ":40000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, unbind)
	if rec.Code == http.StatusOK {
		t.Fatalf("retired unbind path still succeeds")
	}
	if n, _ := st.CountDevices(u.ID); n != 1 {
		t.Errorf("device roster changed through retired public path: %d devices", n)
	}
}
