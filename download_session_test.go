package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

var _ = flag.Bool("validate", false, "accept resumed downloader validation")

func TestResumeDownloadArgsEnablesValidation(t *testing.T) {
	original := []string{"-app", "123"}
	resumed := resumeDownloadArgs(original)
	if len(resumed) != 3 || resumed[2] != "-validate" || len(original) != 2 {
		t.Fatalf("unexpected resume arguments: %v (original %v)", resumed, original)
	}
	if len(resumeDownloadArgs(resumed)) != len(resumed) {
		t.Fatal("validation was added twice")
	}
}

func TestDownloaderInheritedPipeHelper(t *testing.T) {
	switch os.Getenv("DEPOTDROP_PIPE_HELPER") {
	case "parent":
		child := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 4")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		fmt.Println("25.00% partial-file")
		time.Sleep(time.Minute)
		os.Exit(0)
	}
}

func TestCancelWithInheritedOutputPipe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("requires PowerShell")
	}
	t.Setenv("DEPOTDROP_PIPE_HELPER", "parent")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan DownloadUpdate, 8)
	done := make(chan DownloadUpdate, 1)
	go func() {
		done <- downloadAttempt(ctx, os.Args[0], []string{"-test.run=^TestDownloaderInheritedPipeHelper$"}, "", func(update DownloadUpdate) { updates <- update })
	}()
	for {
		select {
		case update := <-updates:
			if update.Progress == 25 {
				cancel()
				goto waiting
			}
		case <-time.After(10 * time.Second):
			t.Fatal("downloader did not start")
		}
	}
waiting:
	select {
	case update := <-done:
		if update.State != "cancelled" {
			t.Fatalf("got %s", update.State)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel waited for a descendant holding the output pipe")
	}
}

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
