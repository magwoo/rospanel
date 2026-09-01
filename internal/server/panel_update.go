package server

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/AppsGanin/rospanel/internal/backup"
	"github.com/AppsGanin/rospanel/internal/updater"
	"github.com/AppsGanin/rospanel/internal/version"
)

// checkUpdate reports the running version and whether a newer release exists in
// this build's fixed update channel.
func (rt *Router) checkUpdate(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"current": version.Version, "available": false}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	rel, err := updater.Latest(ctx, updater.Repo)
	if err != nil {
		resp["error"] = err.Error()
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp["latest"] = rel.Version
	resp["notes"] = rel.Notes
	resp["available"] = rel.AssetURL != "" && updater.IsNewer(rel.Version, version.Version)
	writeJSON(w, http.StatusOK, resp)
}

// applyUpdate downloads the latest release, snapshots the DB, atomically swaps the
// running binary, then schedules a service restart so systemd re-execs it. The
// restart briefly drops Xray (all connections) — the client polls back to life.
func (rt *Router) applyUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	rel, err := updater.Latest(ctx, updater.Repo)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if !updater.IsNewer(rel.Version, version.Version) {
		writeErrCode(w, http.StatusBadRequest, "err.alreadyLatest", "уже установлена последняя версия")
		return
	}
	backupFn := func() error {
		_ = rt.mgr.Store().Checkpoint()
		return backup.Create(rt.dataDir, filepath.Join(rt.dataDir, "pre-update-backup.tgz"))
	}
	if err := updater.Apply(context.Background(), rel, backupFn); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": rel.Version})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	updater.Restart()
}
