package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PickManifestFile lets the user choose any manifest. Downgrade reads the IDs from inside the file,
// so the filename does not have to follow the DEPOTID_MANIFESTID pattern.
func (a *App) PickManifestFile(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: []runtime.FileFilter{{DisplayName: "Steam manifest (*.manifest)", Pattern: "*.manifest"}},
	})
}

// ManifestIdentity is what a manifest says about itself, used to fill in the depot ID after a file is chosen.
type ManifestIdentity struct {
	DepotID    string `json:"depotId"`
	ManifestID string `json:"manifestId"`
}

// InspectDowngradeManifest reads the depot and manifest IDs from inside a manifest file.
func (a *App) InspectDowngradeManifest(path string) (ManifestIdentity, error) {
	manifest, err := readManifestIdentity(filepath.Clean(strings.TrimSpace(path)))
	if err != nil {
		return ManifestIdentity{}, fmt.Errorf("that file %w", err)
	}
	return ManifestIdentity{DepotID: strconv.FormatUint(uint64(manifest.DepotID), 10), ManifestID: strconv.FormatUint(manifest.ManifestID, 10)}, nil
}

// PreflightDowngrade checks everything and reports what a downgrade would do, without touching the game folder.
func (a *App) PreflightDowngrade(request DowngradeRequest) (DowngradeSummary, error) {
	plan, err := planDowngrade(request)
	if err != nil {
		return DowngradeSummary{}, err
	}
	return plan.summary, nil
}

func (a *App) StartDowngrade(request DowngradeRequest) error {
	plan, err := planDowngrade(request)
	if err != nil {
		return err
	}

	// One job at a time: a download and a downgrade can never write to the same game folder together.
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("a download or downgrade is already running")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	a.running = true
	a.paused = false
	a.resume = make(chan struct{}, 1)
	attemptCtx, pauseCancel := context.WithCancelCause(ctx)
	a.pauseCancel = pauseCancel
	a.mu.Unlock()

	emit := func(update DownloadUpdate) { runtime.EventsEmit(a.ctx, "download:update", update) }
	go a.runSupervised(ctx, attemptCtx, func(attemptCtx context.Context, emit func(DownloadUpdate)) DownloadUpdate {
		return downgradeAttempt(attemptCtx, plan, emit)
	}, func(update DownloadUpdate) DownloadUpdate {
		return finishDowngrade(plan, update)
	}, emit)
	return nil
}

// downgradeAttempt runs the downloader in two passes per attempt.
//
// Pass 1 moves the folder from the current version to the target: the downloader reuses every chunk that both
// versions share and fetches the rest. Pass 2 then marks the target as installed and runs a plain validated
// download of the target, exactly like the Download tab. It trusts nothing from pass 1: every file is checked
// against the target manifest and any leftover difference is repaired, so the game never needs a manual
// re-validate. A resumed attempt starts again from pass 1; matching files are skipped quickly.
func downgradeAttempt(ctx context.Context, plan *downgradePlan, emit func(DownloadUpdate)) DownloadUpdate {
	if err := stageDowngrade(plan); err != nil {
		return DownloadUpdate{State: "error", Message: "Could not prepare the downgrade: " + err.Error()}
	}
	commands := compatArguments(plan)
	passes := 2
	steps := float64(len(commands) * passes)
	var elapsed float64
	var downloaded, uncompressed int64
	last := DownloadUpdate{State: "complete", Progress: 100, Message: "Downgrade downloaded"}
	step := 0
	for pass := 0; pass < passes; pass++ {
		if pass == 1 {
			if err := markTargetInstalled(plan); err != nil {
				return DownloadUpdate{State: "error", Message: "Could not prepare the final check: " + err.Error()}
			}
		}
		for _, args := range commands {
			offset := float64(step) * 100 / steps
			scale := 1 / steps
			message := "Downgrading files"
			if pass == 1 {
				message = "Checking every file"
			}
			last = downloadAttempt(ctx, plan.request.DownloaderPath, args, filepath.Dir(plan.request.DownloaderPath), func(update DownloadUpdate) {
				update.Progress = offset + update.Progress*scale
				if update.State == "running" {
					update.Message = message
				}
				emit(update)
			})
			elapsed += last.ElapsedSeconds
			downloaded += last.DownloadedBytes
			uncompressed += last.UncompressedBytes
			if last.State != "complete" {
				last.Progress = offset + last.Progress*scale
				last.ElapsedSeconds, last.DownloadedBytes, last.UncompressedBytes = elapsed, downloaded, uncompressed
				return last
			}
			step++
		}
	}
	last.ElapsedSeconds, last.DownloadedBytes, last.UncompressedBytes = elapsed, downloaded, uncompressed
	last.Progress = 100
	return last
}

// finishDowngrade runs only after every depot downloaded successfully: it confirms the target files are
// right, then removes the files that belonged to the old version only.
func finishDowngrade(plan *downgradePlan, update DownloadUpdate) DownloadUpdate {
	trimOversizeFiles(plan)
	if problem := verifyTargetFiles(plan); problem != "" {
		update.State, update.Message = "error", "Downgrade stopped before removing anything: "+problem
		return update
	}
	removed, failed := removeStaleFiles(plan)
	switch {
	case plan.summary.Partial:
		update.Message = "Downgrade complete for the selected depots. Nothing was removed because other depots share this folder."
	case failed > 0:
		update.Message = fmt.Sprintf("Downgrade complete. Removed %d old files; %d could not be removed.", removed, failed)
	default:
		update.Message = fmt.Sprintf("Downgrade complete. Removed %d old files.", removed)
	}
	return update
}

// verifyTargetFiles checks that every file of the older version exists at exactly its manifest size.
func verifyTargetFiles(plan *downgradePlan) string {
	for _, file := range plan.targets {
		if file.linked {
			continue
		}
		full, err := resolveInside(plan.root, file.rel)
		if err != nil {
			return err.Error()
		}
		stat, err := os.Stat(full)
		if err != nil {
			return file.rel + " is missing"
		}
		if uint64(stat.Size()) != file.size {
			return fmt.Sprintf("%s is %d bytes but should be %d", file.rel, stat.Size(), file.size)
		}
	}
	return ""
}

func removeStaleFiles(plan *downgradePlan) (removed, failed int) {
	for _, file := range plan.removals {
		full, err := resolveInside(plan.root, file.rel)
		if err != nil {
			failed++
			continue
		}
		if stat, err := os.Lstat(full); err != nil || stat.IsDir() {
			continue
		}
		if err := os.Remove(full); err != nil {
			failed++
			continue
		}
		removed++
	}
	return removed, failed
}
