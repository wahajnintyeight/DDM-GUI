package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestUpdateAsset(t *testing.T) {
	assets := []github.ReleaseAsset{{Name: "other.exe"}, {Name: "DDM-GUI.exe"}}
	if got := updateAsset(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets); got != 1 {
		t.Fatalf("Windows asset index = %d", got)
	}
	if got := updateAsset(updater.CheckRequest{Platform: "linux", Arch: "amd64"}, assets); got != -1 {
		t.Fatalf("Linux asset index = %d", got)
	}
}

func TestUpdateRequiresChecksum(t *testing.T) {
	checksum := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/checksum") {
			fmt.Fprint(w, checksum)
			return
		}
		fmt.Fprintf(w, `{"tag_name":"v0.2.5","assets":[{"name":"DDM-GUI.exe","url":%q},{"name":"SHA256SUMS","url":%q}]}`, "https://example.com/app", "http://"+r.Host+"/checksum")
	}))
	defer server.Close()
	p, err := github.New(github.Config{Repository: "owner/repo", BaseURL: server.URL, ChecksumAsset: "SHA256SUMS", AssetMatcher: updateAsset})
	if err != nil {
		t.Fatal(err)
	}
	req := updater.CheckRequest{CurrentVersion: "0.2.4", Platform: "windows", Arch: "amd64"}
	if _, err = (checkedProvider{p}).Check(context.Background(), req); err == nil {
		t.Fatal("missing checksum accepted")
	}
	checksum = strings.Repeat("a", 64) + "  DDM-GUI.exe\n"
	if rel, err := (checkedProvider{p}).Check(context.Background(), req); err != nil || rel == nil {
		t.Fatalf("valid checksum rejected: %v", err)
	}
}

func TestAppVersion(t *testing.T) {
	if version, err := appVersion(); err != nil || version == "" {
		t.Fatalf("version %q: %v", version, err)
	}
}

func TestRestartForUpdateWaitsForRunningJob(t *testing.T) {
	app := &App{running: true}
	app.updater = updater.New(updateHost{app})
	if err := app.RestartForUpdate(); err == nil || !strings.Contains(err.Error(), "finish or cancel") {
		t.Fatalf("restart during a job: %v", err)
	}
}
