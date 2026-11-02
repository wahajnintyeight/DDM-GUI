package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type testFile struct {
	name  string
	size  uint64
	flags uint32
}

func pbVarint(field int, value uint64) []byte {
	return binary.AppendUvarint(binary.AppendUvarint(nil, uint64(field<<3)), value)
}

func pbBytes(field int, value []byte) []byte {
	out := binary.AppendUvarint(nil, uint64(field<<3|2))
	out = binary.AppendUvarint(out, uint64(len(value)))
	return append(out, value...)
}

func section(magic uint32, body []byte) []byte {
	out := binary.LittleEndian.AppendUint32(nil, magic)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(body)))
	return append(out, body...)
}

func encryptName(t *testing.T, key []byte, name string) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := []byte("0123456789abcdef")
	encIV := make([]byte, aes.BlockSize)
	block.Encrypt(encIV, iv)
	pad := aes.BlockSize - len(name)%aes.BlockSize
	plain := append([]byte(name), make([]byte, pad)...)
	for i := len(name); i < len(plain); i++ {
		plain[i] = byte(pad)
	}
	cipherText := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(cipherText, plain)
	return base64.StdEncoding.EncodeToString(append(encIV, cipherText...))
}

func buildManifest(t *testing.T, depot uint32, id uint64, files []testFile, key []byte) []byte {
	t.Helper()
	var payload []byte
	for _, file := range files {
		name := file.name
		if key != nil {
			name = encryptName(t, key, name)
		}
		mapping := append(pbBytes(1, []byte(name)), pbVarint(2, file.size)...)
		mapping = append(mapping, pbVarint(3, uint64(file.flags))...)
		payload = append(payload, pbBytes(1, mapping)...)
	}
	metadata := append(pbVarint(1, uint64(depot)), pbVarint(2, id)...)
	if key != nil {
		metadata = append(metadata, pbVarint(4, 1)...)
	}
	out := section(manifestPayloadMagic, payload)
	out = append(out, section(manifestMetadataMagic, metadata)...)
	out = append(out, section(manifestSignatureMagic, nil)...)
	return binary.LittleEndian.AppendUint32(out, manifestEndMagic)
}

type downgradeFixture struct {
	root, downloader, keys string
	request                DowngradeRequest
}

func newDowngradeFixture(t *testing.T, current, target []testFile, encrypt bool) downgradeFixture {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "game")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	keys := filepath.Join(dir, "keys.txt")
	if err := os.WriteFile(keys, []byte("999;aa\n3468651;"+hex.EncodeToString(key)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	downloader := filepath.Join(dir, "DepotDownloaderMod.exe")
	if err := os.WriteFile(downloader, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	var fileKey []byte
	if encrypt {
		fileKey = key
	}
	currentPath := filepath.Join(dir, "3468651_200.manifest")
	targetPath := filepath.Join(dir, "3468651_100.manifest")
	if err := os.WriteFile(currentPath, buildManifest(t, 3468651, 200, current, fileKey), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, buildManifest(t, 3468651, 100, target, fileKey), 0o600); err != nil {
		t.Fatal(err)
	}
	return downgradeFixture{root: root, downloader: downloader, keys: keys, request: DowngradeRequest{
		AppID: "3468650", GameFolder: root, DownloaderPath: downloader, KeysPath: keys,
		Depots: []DowngradeDepot{{DepotID: "3468651", CurrentManifestPath: currentPath, TargetManifestPath: targetPath}},
	}}
}

func writeGameFile(t *testing.T, root, rel string, size int) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func TestParseManifestPlainAndEncrypted(t *testing.T) {
	files := []testFile{{name: `Data\a.pak`, size: 10}, {name: "Bin", flags: manifestFlagDirectory}}
	for _, encrypt := range []bool{false, true} {
		fixture := newDowngradeFixture(t, files, files, encrypt)
		key, _ := depotKeyFor(fixture.keys, "3468651")
		manifest, err := loadManifest(fixture.request.Depots[0].CurrentManifestPath, key)
		if err != nil {
			t.Fatalf("encrypt=%v: %v", encrypt, err)
		}
		if manifest.DepotID != 3468651 || manifest.ManifestID != 200 || len(manifest.Files) != 2 || manifest.Files[0].Name != `Data\a.pak` || !manifest.Files[1].IsDirectory() {
			t.Fatalf("encrypt=%v: unexpected manifest %+v", encrypt, manifest)
		}
	}
	fixture := newDowngradeFixture(t, files, files, true)
	if _, err := loadManifest(fixture.request.Depots[0].CurrentManifestPath, nil); err == nil || !strings.Contains(err.Error(), "depot key") {
		t.Fatalf("encrypted manifest without key: %v", err)
	}
	wrong := make([]byte, 32)
	if _, err := loadManifest(fixture.request.Depots[0].CurrentManifestPath, wrong); err == nil {
		t.Fatal("wrong key was accepted")
	}
	garbage := filepath.Join(t.TempDir(), "1_2.manifest")
	_ = os.WriteFile(garbage, []byte("not a manifest at all"), 0o600)
	if _, err := loadManifest(garbage, nil); err == nil {
		t.Fatal("garbage was accepted as a manifest")
	}
}

func TestPlanRemovesOnlyCurrentOnlyFilesPresentOnDisk(t *testing.T) {
	current := []testFile{{name: "keep.bin", size: 5}, {name: `Old\gone.dat`, size: 7}, {name: "also-gone.dat", size: 3}, {name: "never-installed.dat", size: 1}}
	target := []testFile{{name: "KEEP.BIN", size: 4}, {name: "new.bin", size: 9}}
	fixture := newDowngradeFixture(t, current, target, true)
	writeGameFile(t, fixture.root, "keep.bin", 5)
	writeGameFile(t, fixture.root, "Old/gone.dat", 7)
	writeGameFile(t, fixture.root, "also-gone.dat", 3)
	writeGameFile(t, fixture.root, "unrelated/save.dat", 11)

	plan, err := planDowngrade(fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.summary.RemoveCount != 2 || plan.summary.RemoveBytes != 10 {
		t.Fatalf("summary = %+v", plan.summary)
	}
	if got := plan.summary.Depots[0]; got.CurrentManifestID != "200" || got.TargetManifestID != "100" {
		t.Fatalf("manifest ids = %+v", got)
	}
	if plan.summary.HadStateFolder {
		t.Fatal("reported a .DepotDownloader folder that does not exist")
	}
	if len(downgradeArguments(plan)) != 1 {
		t.Fatal("expected one downloader invocation")
	}
}

func TestPlanRejectsBadInputBeforeTouchingTheGame(t *testing.T) {
	files := []testFile{{name: "a.bin", size: 1}}
	cases := map[string]func(*downgradeFixture){
		"depot mismatch": func(f *downgradeFixture) { f.request.Depots[0].DepotID = "999" },
		"missing key":    func(f *downgradeFixture) { _ = os.WriteFile(f.keys, []byte("999;aa\n"), 0o600) },
		"same manifest": func(f *downgradeFixture) {
			f.request.Depots[0].TargetManifestPath = f.request.Depots[0].CurrentManifestPath
		},
		"missing target": func(f *downgradeFixture) {
			f.request.Depots[0].TargetManifestPath = filepath.Join(f.root, "nope.manifest")
		},
		"no game folder":  func(f *downgradeFixture) { f.request.GameFolder = filepath.Join(f.root, "missing") },
		"bad app id":      func(f *downgradeFixture) { f.request.AppID = "abc" },
		"duplicate depot": func(f *downgradeFixture) { f.request.Depots = append(f.request.Depots, f.request.Depots[0]) },
		"swapped file names": func(f *downgradeFixture) {
			d := &f.request.Depots[0]
			d.CurrentManifestPath, d.TargetManifestPath = d.TargetManifestPath, d.CurrentManifestPath
			// Contents still match their filenames, so rename one over the other to break that.
			_ = os.Rename(d.CurrentManifestPath, filepath.Join(filepath.Dir(d.CurrentManifestPath), "3468651_100x.tmp"))
			_ = os.Link(d.TargetManifestPath, d.CurrentManifestPath)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			fixture := newDowngradeFixture(t, files, files, true)
			mutate(&fixture)
			if _, err := planDowngrade(fixture.request); err == nil {
				t.Fatal("expected a preflight error")
			}
			entries, _ := os.ReadDir(fixture.root)
			if len(entries) != 0 {
				t.Fatal("preflight wrote to the game folder")
			}
		})
	}
}

func TestPlanRejectsUnsafeManifestPaths(t *testing.T) {
	for _, name := range []string{`..\outside.dll`, "../outside.dll", "a/../../outside.dll", "/abs.dll", `C:\Windows\x.dll`, "stream.txt:hidden", ""} {
		fixture := newDowngradeFixture(t, []testFile{{name: "a.bin", size: 1}}, []testFile{{name: name, size: 1}}, false)
		if _, err := planDowngrade(fixture.request); err == nil {
			t.Fatalf("path %q was accepted", name)
		}
		fixture = newDowngradeFixture(t, []testFile{{name: name, size: 1}}, []testFile{{name: "a.bin", size: 1}}, false)
		if _, err := planDowngrade(fixture.request); err == nil {
			t.Fatalf("current-manifest path %q was accepted", name)
		}
	}
	if rel, err := normalizeManifestPath(`Data\.\sub\file.pak`); err != nil || rel != "Data/sub/file.pak" {
		t.Fatalf("normalize = %q, %v", rel, err)
	}
}

func TestPlanRequiresOtherInstalledDepots(t *testing.T) {
	files := []testFile{{name: "a.bin", size: 1}}
	fixture := newDowngradeFixture(t, files, files[:0], false)
	state := filepath.Join(fixture.root, ".DepotDownloader")
	_ = os.MkdirAll(state, 0o755)
	if err := writeDepotConfig(filepath.Join(state, "depot.config"), map[uint32]uint64{3468651: 200, 3468652: 55}); err != nil {
		t.Fatal(err)
	}
	writeGameFile(t, fixture.root, "a.bin", 1)

	if _, err := planDowngrade(fixture.request); err == nil || !strings.Contains(err.Error(), "3468652") {
		t.Fatalf("other depot not required: %v", err)
	}
	fixture.request.AcknowledgePartial = true
	plan, err := planDowngrade(fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.summary.Partial || plan.summary.RemoveCount != 0 || !plan.summary.HadStateFolder {
		t.Fatalf("partial summary = %+v", plan.summary)
	}
	update := finishDowngrade(plan, DownloadUpdate{State: "complete"})
	if update.State != "complete" || !exists(fixture.root, "a.bin") {
		t.Fatalf("partial downgrade removed files: %+v", update)
	}
}

func TestFinishDowngradeVerifiesThenRemoves(t *testing.T) {
	current := []testFile{{name: "keep.bin", size: 5}, {name: "gone.dat", size: 7}}
	target := []testFile{{name: "keep.bin", size: 4}}
	fixture := newDowngradeFixture(t, current, target, false)
	writeGameFile(t, fixture.root, "keep.bin", 3) // too short for the target size
	writeGameFile(t, fixture.root, "gone.dat", 7)
	writeGameFile(t, fixture.root, "unrelated.sav", 2)
	plan, err := planDowngrade(fixture.request)
	if err != nil {
		t.Fatal(err)
	}

	update := finishDowngrade(plan, DownloadUpdate{State: "complete"})
	if update.State != "error" || !strings.Contains(update.Message, "keep.bin") {
		t.Fatalf("wrong size was not caught: %+v", update)
	}
	if !exists(fixture.root, "gone.dat") {
		t.Fatal("stale file was removed even though the target was not right")
	}

	writeGameFile(t, fixture.root, "keep.bin", 6) // too long: cut back to the target size
	update = finishDowngrade(plan, DownloadUpdate{State: "complete"})
	if update.State != "complete" || !strings.Contains(update.Message, "Removed 1") {
		t.Fatalf("finish = %+v", update)
	}
	if exists(fixture.root, "gone.dat") || !exists(fixture.root, "keep.bin") || !exists(fixture.root, "unrelated.sav") {
		t.Fatal("cleanup removed the wrong files")
	}
}

func TestDowngradeArgumentsAreOptInAndNormalDownloadIsUnchanged(t *testing.T) {
	files := []testFile{{name: "a.bin", size: 1}}
	fixture := newDowngradeFixture(t, files, files, false)
	plan, err := planDowngrade(fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(downgradeArguments(plan)[0], " ")
	for _, want := range []string{"-sourcemanifestfile " + fixture.request.Depots[0].CurrentManifestPath, "-manifestfile " + fixture.request.Depots[0].TargetManifestPath, "-manifest " + strconv.Itoa(100), "-validate"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args %q lack %q", args, want)
		}
	}
	info := ManifestInfo{Path: fixture.request.Depots[0].TargetManifestPath, DepotID: "3468651", ManifestID: "100"}
	_, normal, err := downloadCommand(info, fixture.root, fixture.downloader, fixture.keys, "3468650", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(normal, " "), downgradeFlag) {
		t.Fatal("the normal download command picked up the downgrade option")
	}
}

func TestKeysNeverLeakIntoErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.txt")
	_ = os.WriteFile(path, []byte("\xef\xbb\xbf5;SECRETSECRET\n"), 0o600)
	_, err := depotKeyFor(path, "5")
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("error = %v", err)
	}
}

// Opt-in check against a real downloader build: DEPOTDROP_REAL_DOWNLOADER=path\to\DepotDownloaderMod.exe
func TestRealDownloaderAdvertisesDowngrade(t *testing.T) {
	path := os.Getenv("DEPOTDROP_REAL_DOWNLOADER")
	if path == "" {
		t.Skip("set DEPOTDROP_REAL_DOWNLOADER to test a real downloader build")
	}
	supported, err := downloaderSupportsDowngrade(path)
	if err != nil || !supported {
		t.Fatalf("supported=%v err=%v", supported, err)
	}
}

func TestReadManifestIdentityNeedsNoKey(t *testing.T) {
	files := []testFile{{name: "a.bin", size: 1}}
	fixture := newDowngradeFixture(t, files, files, true) // filenames encrypted
	identity, err := readManifestIdentity(fixture.request.Depots[0].CurrentManifestPath)
	if err != nil || identity.DepotID != 3468651 || identity.ManifestID != 200 {
		t.Fatalf("identity = %+v, %v", identity, err)
	}
	garbage := filepath.Join(t.TempDir(), "x.manifest")
	_ = os.WriteFile(garbage, []byte("nope"), 0o600)
	if _, err := readManifestIdentity(garbage); err == nil {
		t.Fatal("garbage was accepted")
	}
}

func TestStageDowngradeForStockDownloader(t *testing.T) {
	for _, encrypt := range []bool{false, true} {
		current := []testFile{{name: "keep.bin", size: 5}, {name: "gone.dat", size: 7}}
		target := []testFile{{name: "keep.bin", size: 4}}
		fixture := newDowngradeFixture(t, current, target, encrypt)
		other := filepath.Join(fixture.root, ".DepotDownloader")
		_ = os.MkdirAll(other, 0o755)
		_ = writeDepotConfig(filepath.Join(other, "depot.config"), map[uint32]uint64{3468651: 1, 3468652: 55})
		plan, err := planDowngrade(DowngradeRequest{AppID: fixture.request.AppID, GameFolder: fixture.root, DownloaderPath: fixture.downloader, KeysPath: fixture.keys, Depots: fixture.request.Depots, AcknowledgePartial: true})
		if err != nil {
			t.Fatal(err)
		}
		if err := stageDowngrade(plan); err != nil {
			t.Fatal(err)
		}
		installed, err := readDepotConfig(filepath.Join(other, "depot.config"))
		if err != nil || installed[3468651] != 200 || installed[3468652] != 55 {
			t.Fatalf("depot.config = %v, %v", installed, err)
		}
		staged := func(id string) *DepotManifest {
			path := filepath.Join(other, "3468651_"+id+".manifest")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sum, _ := os.ReadFile(path + ".sha")
			if len(sum) != 20 {
				t.Fatalf("sha file has %d bytes", len(sum))
			}
			manifest, err := parseManifest(data, nil) // no key: names must already be plain
			if err != nil {
				t.Fatal(err)
			}
			return manifest
		}
		trimmed, full := staged("200"), staged("100")
		if len(trimmed.Files) != 1 || trimmed.Files[0].Name != "keep.bin" || trimmed.ManifestID != 200 {
			t.Fatalf("current copy = %+v", trimmed)
		}
		if len(full.Files) != 1 || full.Files[0].Name != "keep.bin" || full.Files[0].Size != 4 {
			t.Fatalf("target copy = %+v", full)
		}
		args := strings.Join(compatArguments(plan)[0], " ")
		if strings.Contains(args, "-manifestfile") || strings.Contains(args, downgradeFlag) || !strings.Contains(args, "-validate") {
			t.Fatalf("args = %s", args)
		}
	}
}
