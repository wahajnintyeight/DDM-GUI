package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspectManifestUsesLastFilenameNumber(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "3468651_795493147105120679.manifest")
	if err := os.WriteFile(path, []byte("manifest"), 0o600); err != nil {
		t.Fatal(err)
	}

	info, err := inspectManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.DepotID != "3468651" || info.ManifestID != "795493147105120679" {
		t.Fatalf("unexpected parsed IDs: %#v", info)
	}
}

func TestDownloadCommandMatchesBundleContract(t *testing.T) {
	directory := t.TempDir()
	manifestPath := filepath.Join(directory, "3468651_795493147105120679.manifest")
	for _, path := range []string{manifestPath, filepath.Join(directory, "DepotDownloaderMod.exe"), filepath.Join(directory, "keys.txt")} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gameFolder := filepath.Join(directory, "Cricket 26")
	if err := os.Mkdir(gameFolder, 0o700); err != nil {
		t.Fatal(err)
	}

	info, _ := inspectManifest(manifestPath)
	executable, args, err := downloadCommand(info, gameFolder)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-app", appID, "-depot", "3468651", "-manifest", "795493147105120679", "-manifestfile", manifestPath, "-depotkeys", filepath.Join(directory, "keys.txt"), "-dir", gameFolder, "-validate"}
	if executable != filepath.Join(directory, "DepotDownloaderMod.exe") || !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected command: %q %#v", executable, args)
	}
}

func TestParseProgress(t *testing.T) {
	progress, ok := parseProgress(`99.73% E:\Games\Cricket 26\data\file`)
	if !ok || progress != 99.73 {
		t.Fatalf("got %v, %v", progress, ok)
	}
	if _, ok := parseProgress("Depot disconnected"); ok {
		t.Fatal("parsed progress from a status line")
	}
}
