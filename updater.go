package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

//go:embed wails.json
var versionConfig embed.FS

type updateHost struct{ app *App }

func (h updateHost) Emit(name string, data ...any) bool {
	runtime.EventsEmit(h.app.ctx, name, data...)
	return false
}
func (h updateHost) OnEvent(string, func(any)) func()                      { return func() {} }
func (h updateHost) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }
func (h updateHost) Quit()                                                 { runtime.Quit(h.app.ctx) }

type checkedProvider struct{ *github.Provider }

func (p checkedProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	rel, err := p.Provider.Check(ctx, req)
	if err == nil && rel != nil && (rel.Verification == nil || len(rel.Verification.Digest) != 32) {
		return nil, errors.New("this release has no valid SHA256SUMS entry; please try again later")
	}
	return rel, err
}

func updateAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	if req.Platform != "windows" || req.Arch != "amd64" {
		return -1
	}
	for i, asset := range assets {
		if asset.Name == "DDM-GUI.exe" {
			return i
		}
	}
	return -1
}

func appVersion() (string, error) {
	data, err := versionConfig.ReadFile("wails.json")
	if err != nil {
		return "", err
	}
	var config struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err = json.Unmarshal(data, &config); err != nil {
		return "", err
	}
	if config.Info.ProductVersion == "" {
		return "", errors.New("app version is missing")
	}
	return config.Info.ProductVersion, nil
}

func (a *App) initUpdater() error {
	if a.updater != nil {
		return nil
	}
	version, err := appVersion()
	if err != nil {
		return err
	}
	provider, err := github.New(github.Config{Repository: "wahajnintyeight/DDM-GUI", ChecksumAsset: "SHA256SUMS", AssetMatcher: updateAsset})
	if err != nil {
		return err
	}
	u := updater.New(updateHost{a})
	if err := u.Init(updater.Config{CurrentVersion: version, Providers: []updater.Provider{checkedProvider{provider}}}); err != nil {
		return err
	}
	a.updater = u
	return nil
}

type UpdateInfo struct {
	Current   string `json:"current"`
	Available string `json:"available"`
}

func (a *App) CheckForUpdate() (UpdateInfo, error) {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	if goruntime.GOOS != "windows" || goruntime.GOARCH != "amd64" {
		return UpdateInfo{}, errors.New("automatic updates are only available on Windows 64-bit")
	}
	if err := a.initUpdater(); err != nil {
		return UpdateInfo{}, err
	}
	rel, err := a.updater.Check(a.ctx)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("could not check for updates: %w", err)
	}
	result := UpdateInfo{Current: a.updater.CurrentVersion()}
	if rel != nil {
		result.Available = rel.Version
	}
	return result, nil
}
func (a *App) DownloadUpdate() error {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	if a.updater == nil {
		return errors.New("check for an update first")
	}
	return a.updater.DownloadAndInstall(a.ctx)
}
func (a *App) RestartForUpdate() error {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	if a.updater == nil {
		return errors.New("no update is ready")
	}
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("finish or cancel your current download or downgrade before restarting")
	}
	a.running = true
	a.mu.Unlock()
	if err := a.updater.Restart(a.ctx); err != nil {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
		return err
	}
	return nil
}
