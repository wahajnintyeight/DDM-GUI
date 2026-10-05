package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// Reuse the test binary as a downloader so the check exercises real process exit.
func TestDownloaderProcess(t *testing.T) {
	if os.Getenv("DEPOTDROP_TEST_DOWNLOADER") != "1" {
		return
	}
	fmt.Println("25.00% partial-file")
	time.Sleep(time.Minute)
	os.Exit(0)
}

func TestPauseBeforeDownloaderStarts(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errDownloadPaused)
	update := downloadAttempt(ctx, "must-not-launch", nil, "", func(DownloadUpdate) {
		t.Fatal("emitted running after pause")
	})
	if update.State != "paused" {
		t.Fatalf("early pause became %s", update.State)
	}
}

func TestDownloadPauseResumeCancel(t *testing.T) {
	t.Setenv("DEPOTDROP_TEST_DOWNLOADER", "1")
	for _, resume := range []bool{false, true} {
		t.Run(fmt.Sprintf("resume=%v", resume), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			attemptCtx, pauseCancel := context.WithCancelCause(ctx)
			a := &App{running: true, cancel: cancel, pauseCancel: pauseCancel, resume: make(chan struct{}, 1)}
			updates := make(chan DownloadUpdate, 32)
			done := make(chan struct{})
			go func() {
				a.runDownload(ctx, attemptCtx, os.Args[0], []string{"-test.run=^TestDownloaderProcess$"}, t.TempDir(), func(update DownloadUpdate) { updates <- update })
				close(done)
			}()
			waitFor := func(state string, progress float64) DownloadUpdate {
				t.Helper()
				timeout := time.NewTimer(10 * time.Second)
				defer timeout.Stop()
				for {
					select {
					case update := <-updates:
						if update.State == state && update.Progress == progress {
							return update
						}
						if update.State == "error" || update.State == "complete" {
							t.Fatalf("unexpected update: %+v", update)
						}
					case <-timeout.C:
						t.Fatalf("timed out waiting for %s", state)
					}
				}
			}
			if err := a.ResumeDownload(); err == nil {
				t.Fatal("resumed an active download")
			}
			waitFor("running", 25)
			if err := a.PauseDownload(); err != nil {
				t.Fatal(err)
			}
			waitFor("paused", 25)
			if err := a.PauseDownload(); err == nil {
				t.Fatal("paused twice")
			}
			if resume {
				if err := a.ResumeDownload(); err != nil {
					t.Fatal(err)
				}
				waitFor("running", 25)
				if err := a.PauseDownload(); err != nil {
					t.Fatal(err)
				}
				waitFor("paused", 25)
			}
			a.CancelDownload()
			waitFor("cancelled", 25)
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("session did not exit")
			}
			if a.running || a.paused || a.cancel != nil || a.pauseCancel != nil {
				t.Fatal("session did not clear its state")
			}
			if err := a.ResumeDownload(); err == nil {
				t.Fatal("resumed a cancelled download")
			}
		})
	}
}
