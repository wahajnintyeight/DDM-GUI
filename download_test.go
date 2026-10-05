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
	downloaderPath := filepath.Join(directory, "DepotDownloaderMod.exe")
	keysPath := filepath.Join(directory, "keys.txt")
	executable, args, err := downloadCommand(info, gameFolder, downloaderPath, keysPath, "3468650", "", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-app", "3468650", "-depot", "3468651", "-manifest", "795493147105120679", "-manifestfile", manifestPath, "-depotkeys", keysPath, "-dir", gameFolder, "-validate"}
	if executable != downloaderPath || !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected command: %q %#v", executable, args)
	}
	_, args, err = downloadCommand(info, gameFolder, downloaderPath, keysPath, "3468650", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, want[:len(want)-1]) {
		t.Fatalf("validation disabled but got arguments: %#v", args)
	}
}

func TestDownloadCommandAllowsDepotOverride(t *testing.T) {
	directory := t.TempDir()
	manifestPath := filepath.Join(directory, "3468651_795493147105120679.manifest")
	downloaderPath := filepath.Join(directory, "downloader.exe")
	keysPath := filepath.Join(directory, "custom-keys.txt")
	gameFolder := filepath.Join(directory, "game")
	for _, path := range []string{manifestPath, downloaderPath, keysPath} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(gameFolder, 0o700); err != nil {
		t.Fatal(err)
	}

	info, err := inspectManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	_, args, err := downloadCommand(info, gameFolder, downloaderPath, keysPath, "123456", "7654321", true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args[:6], []string{"-app", "123456", "-depot", "7654321", "-manifest", "795493147105120679"}) {
		t.Fatalf("unexpected overridden IDs: %#v", args)
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

func TestParseDownloadTotals(t *testing.T) {
	downloaded, uncompressed, ok := parseDownloadTotals("Total downloaded: 1,737,996,624 bytes (1,770,372,614 bytes uncompressed) from 1 depots")
	if !ok || downloaded != 1737996624 || uncompressed != 1770372614 {
		t.Fatalf("got %d, %d, %v", downloaded, uncompressed, ok)
	}
	if _, _, ok := parseDownloadTotals("Disconnected from Steam"); ok {
		t.Fatal("parsed totals from a status line")
	}
}
