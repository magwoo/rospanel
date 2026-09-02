package core

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/AppsGanin/rospanel/internal/actor"
	"github.com/AppsGanin/rospanel/internal/i18n"
	"github.com/AppsGanin/rospanel/internal/model"
)

// Device binding. A client that follows the subscription-header convention sends a
// stable install id in x-hwid; the panel binds it to the user on first fetch and
// refuses the fetch once the user's device cap is full. See migration 0041 for why
// this exists next to the IP-based count rather than replacing it.

// deviceRefusalQuiet is how long the same (user, device) refusal stays silent after
// it has been reported once. A refused client keeps retrying on its own update
// timer — without this, one person who installed the app on a fourth phone would
// write an audit row and ping the operator every few minutes, forever.
const deviceRefusalQuiet = 6 * time.Hour

type deviceNotice struct {
	mu    sync.Mutex
	quiet time.Duration
	seen  map[string]time.Time
}

func newDeviceNotice() *deviceNotice { return newNotice(deviceRefusalQuiet) }

func newNotice(quiet time.Duration) *deviceNotice {
	return &deviceNotice{quiet: quiet, seen: map[string]time.Time{}}
}

func (n *deviceNotice) should(key string, now time.Time) bool {
	if n == nil {
		return true
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if last, ok := n.seen[key]; ok && now.Sub(last) < n.quiet {
		return false
	}
	for k, t := range n.seen {
		if now.Sub(t) >= n.quiet {
			delete(n.seen, k)
		}
	}
	n.seen[key] = now
	return true
}

type DeviceVerdict struct {
	Allow bool
	Cap   int
	Count int
}

func (m *Manager) AdmitDevice(ctx context.Context, u model.User, set *model.Settings, d model.Device) DeviceVerdict {
	if !set.HWIDEnabled {
		return DeviceVerdict{Allow: true}
	}
	if d.HWID == "" {
		return DeviceVerdict{Allow: !set.HWIDRequire}
	}
	capacity := set.DeviceCap(u)
	adm, err := m.store.RegisterDevice(u.ID, d, capacity)
	if err != nil {
		logErr("devices: register failed", "user", u.ID, "err", err)
		return DeviceVerdict{Allow: true}
	}
	switch {
	case adm.New:
		m.audit(ctx, u.ID, model.EventDeviceBound, map[string]any{
			"hwid": d.HWID, "os": d.OS, "model": d.Model,
			"devices": adm.Count, "device_limit": capacity,
		})
	case !adm.Allowed:
		m.reportDeviceRefusal(ctx, u, set, d, capacity, adm.Count)
	}
	return DeviceVerdict{Allow: adm.Allowed, Cap: capacity, Count: adm.Count}
}

func (m *Manager) reportDeviceRefusal(
	ctx context.Context, u model.User, set *model.Settings, d model.Device, capacity, count int,
) {
	if !m.devNotice.should(deviceKey(u.ID), time.Now()) {
		return
	}
	m.audit(ctx, u.ID, model.EventDeviceRefused, map[string]any{
		"hwid": d.HWID, "os": d.OS, "model": d.Model,
		"devices": count, "device_limit": capacity,
	})
	m.notifyAdminEvent(model.AdminEventDeviceLimited, fmt.Sprintf(
		i18n.T(m.botLang(), "notify.adminDeviceRefused"),
		escHTML(u.Name), escHTML(deviceLabel(d)), count, capacity))
	m.notifyUserEvent(set, u, model.UserNotifyDeviceLimited, fmt.Sprintf(
		i18n.T(m.userLang(u.TgChatID), "notify.userDeviceRefused"), count, capacity))
	m.EmitWebhook(model.WebhookUserDeviceLimit, userEventData(u))
}

func (m *Manager) UserDevices(userID int64) ([]model.Device, error) {
	return m.store.ListDevices(userID)
}

func (m *Manager) DeviceCount(userID int64) int {
	n, err := m.store.CountDevices(userID)
	if err != nil {
		logErr("devices: count failed", "user", userID, "err", err)
		return 0
	}
	return n
}

// UnbindDevice is operator-only on this fork. A subscription token authenticates a
// user to fetch their profile, but it must not grant access to the device roster or
// let the holder free device slots. Admin/API/Telegram actors keep the upstream
// behavior unchanged.
func (m *Manager) UnbindDevice(ctx context.Context, userID int64, hwid string) (bool, error) {
	if actor.From(ctx).Kind == model.ActorUser {
		return false, nil
	}
	ok, err := m.store.DeleteDevice(userID, hwid)
	if err != nil || !ok {
		return ok, err
	}
	m.audit(ctx, userID, model.EventDeviceUnbound, map[string]any{"hwid": hwid})
	return true, nil
}

func (m *Manager) UnbindAllDevices(ctx context.Context, userID int64) (int64, error) {
	n, err := m.store.DeleteDevices(userID)
	if err != nil || n == 0 {
		return n, err
	}
	m.audit(ctx, userID, model.EventDeviceUnbound, map[string]any{"devices": n})
	return n, nil
}

func (m *Manager) PurgeIdleDevices() {
	set, err := m.store.GetSettings()
	if err != nil || set.HWIDTTLDays <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -set.HWIDTTLDays).Unix()
	n, err := m.store.PurgeDevices(cutoff)
	if err != nil {
		logErr("devices: retention sweep failed", "err", err)
		return
	}
	if n > 0 {
		logInfo("devices: forgot idle devices", "count", n, "ttl_days", set.HWIDTTLDays)
	}
}

func deviceKey(userID int64) string {
	return strconv.FormatInt(userID, 10)
}

func deviceLabel(d model.Device) string {
	switch {
	case d.Model != "" && d.OS != "":
		return d.Model + " (" + d.OS + ")"
	case d.Model != "":
		return d.Model
	case d.OS != "":
		return d.OS
	default:
		return d.HWID
	}
}

func (m *Manager) SetDeviceCountMode(mode string) error {
	switch mode {
	case model.DeviceCountAuto, model.DeviceCountHWID, model.DeviceCountBoth:
	default:
		return invalidCode("err.badDeviceCountMode",
			"режим подсчёта устройств: {{allowed}}",
			map[string]any{"allowed": "auto, hwid, both"})
	}
	if err := m.store.SetDeviceCountMode(mode); err != nil {
		return err
	}
	m.TriggerUserSync()
	return nil
}
