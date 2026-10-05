package main

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	mu          sync.Mutex
	cancel      context.CancelFunc
	running     bool
	pauseCancel context.CancelCauseFunc
	paused      bool
	resume      chan struct{}
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) SelectManifest() (ManifestInfo, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose a Steam manifest",
		Filters: []runtime.FileFilter{
			{DisplayName: "Steam manifest (*.manifest)", Pattern: "*.manifest"},
		},
	})
	if err != nil || path == "" {
		return ManifestInfo{}, err
	}
	return inspectManifest(path)
}

func (a *App) SelectGameFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose the game folder",
	})
}

func (a *App) SelectDownloader() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose DepotDownloaderMod.exe",
		Filters: []runtime.FileFilter{
			{DisplayName: "DepotDownloaderMod (*.exe)", Pattern: "*.exe"},
		},
	})
}

func (a *App) SelectKeys() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose keys.txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "Depot keys (keys.txt)", Pattern: "keys.txt"},
		},
	})
}
