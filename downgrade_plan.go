package main

import (
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var driveLetterPattern = regexp.MustCompile(`^[A-Za-z]:`)

// DowngradeDepot is one depot to downgrade: the manifest installed now and the older one to restore.
type DowngradeDepot struct {
	DepotID             string `json:"depotId"`
	CurrentManifestPath string `json:"currentManifestPath"`
	TargetManifestPath  string `json:"targetManifestPath"`
}

type DowngradeRequest struct {
	AppID          string           `json:"appId"`
	GameFolder     string           `json:"gameFolder"`
	DownloaderPath string           `json:"downloaderPath"`
	KeysPath       string           `json:"keysPath"`
	Depots         []DowngradeDepot `json:"depots"`
	// AcknowledgePartial lets a single-depot downgrade proceed when the game folder holds other depots.
	// Nothing is removed in that case, because the other depots might own the same files.
	AcknowledgePartial bool `json:"acknowledgePartial"`
}

type DowngradeDepotSummary struct {
	DepotID           string `json:"depotId"`
	CurrentManifestID string `json:"currentManifestId"`
	TargetManifestID  string `json:"targetManifestId"`
	CurrentFiles      int    `json:"currentFiles"`
	TargetFiles       int    `json:"targetFiles"`
}

type DowngradeSummary struct {
	AppID          string                  `json:"appId"`
	GameFolder     string                  `json:"gameFolder"`
	Depots         []DowngradeDepotSummary `json:"depots"`
	RemoveCount    int                     `json:"removeCount"`
	RemoveBytes    int64                   `json:"removeBytes"`
	HadStateFolder bool                    `json:"hadStateFolder"`
	OtherDepots    []string                `json:"otherDepots"`
	Partial        bool                    `json:"partial"`
	Notes          []string                `json:"notes"`
}

type downgradeDepotPlan struct {
	depot   DowngradeDepot
	current *DepotManifest
	target  *DepotManifest
}

type downgradePlan struct {
	request  DowngradeRequest
	root     string
	depots   []downgradeDepotPlan
	targets  []plannedFile // every non-directory file the target version must contain
	removals []plannedFile // files owned by the current version only, present on disk
	summary  DowngradeSummary
}

type plannedFile struct {
	rel    string // forward-slash path relative to the game folder
	size   uint64
	linked bool // a symbolic link entry, which has no content of its own
}

func pathKey(rel string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(rel)
	}
	return rel
}

// normalizeManifestPath turns a manifest filename into a clean relative path, or refuses it.
func normalizeManifestPath(name string) (string, error) {
	if strings.ContainsRune(name, 0) || strings.Contains(name, ":") {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	slashed := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(slashed, "/") || driveLetterPattern.MatchString(slashed) {
		return "", fmt.Errorf("absolute path %q", name)
	}
	var parts []string
	for _, part := range strings.Split(slashed, "/") {
		switch part {
		case "", ".":
		case "..":
			return "", fmt.Errorf("path %q would leave the game folder", name)
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("empty path %q", name)
	}
	return path.Join(parts...), nil
}

// resolveInside joins rel to root and confirms the result, including symlinked parents, stays inside root.
func resolveInside(root, rel string) (string, error) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	if within, err := filepath.Rel(root, full); err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q would leave the game folder", rel)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(full))
	if err != nil {
		if os.IsNotExist(err) {
			return full, nil
		}
		return "", err
	}
	if within, err := filepath.Rel(root, parent); err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q leaves the game folder through a link", rel)
	}
	return full, nil
}

func requireReadableFile(label, p string) (string, error) {
	p = filepath.Clean(strings.TrimSpace(p))
	if p == "." || p == "" {
		return "", fmt.Errorf("choose %s", label)
	}
	stat, err := os.Stat(p)
	if err != nil || stat.IsDir() {
		return "", fmt.Errorf("%s could not be opened at the selected path", label)
	}
	file, err := os.Open(p)
	if err != nil {
		return "", fmt.Errorf("%s could not be read: %w", label, err)
	}
	file.Close()
	return p, nil
}

func checkManifestIdentity(label, p string, manifest *DepotManifest, depotID uint64) error {
	if manifest.DepotID == 0 || manifest.ManifestID == 0 {
		return fmt.Errorf("%s is not a valid manifest (no depot or manifest ID inside)", label)
	}
	if uint64(manifest.DepotID) != depotID {
		return fmt.Errorf("%s belongs to depot %d, not depot %d", label, manifest.DepotID, depotID)
	}
	if parts := manifestNamePattern.FindStringSubmatch(filepath.Base(p)); len(parts) == 3 {
		if parts[1] != strconv.FormatUint(uint64(manifest.DepotID), 10) || parts[2] != strconv.FormatUint(manifest.ManifestID, 10) {
			return fmt.Errorf("%s: the filename says depot %s manifest %s but the file contains depot %d manifest %d", label, parts[1], parts[2], manifest.DepotID, manifest.ManifestID)
		}
	}
	return nil
}

func planDowngrade(request DowngradeRequest) (*downgradePlan, error) {
	appID := strings.TrimSpace(request.AppID)
	if !numericIDPattern.MatchString(appID) {
		return nil, errors.New("enter a numeric game app ID")
	}
	root := filepath.Clean(strings.TrimSpace(request.GameFolder))
	if root == "." || root == "" {
		return nil, errors.New("choose a game folder")
	}
	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return nil, errors.New("the game folder does not exist; choose an existing folder")
	}
	if _, err := os.ReadDir(root); err != nil {
		return nil, fmt.Errorf("the game folder could not be read: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("the game folder could not be resolved: %w", err)
	}
	downloader, err := requireReadableFile("DepotDownloaderMod.exe", request.DownloaderPath)
	if err != nil {
		return nil, err
	}
	keys, err := requireReadableFile("keys.txt", request.KeysPath)
	if err != nil {
		return nil, err
	}
	if len(request.Depots) == 0 {
		return nil, errors.New("add the depot, the current manifest file and the previous version manifest")
	}

	plan := &downgradePlan{request: request, root: resolvedRoot}
	plan.request.AppID, plan.request.GameFolder, plan.request.DownloaderPath, plan.request.KeysPath = appID, resolvedRoot, downloader, keys
	plan.summary = DowngradeSummary{AppID: appID, GameFolder: resolvedRoot}

	seen := map[string]bool{}
	targetSet := map[string]bool{}
	currentFiles := map[string]plannedFile{}
	for _, depot := range request.Depots {
		depotID := strings.TrimSpace(depot.DepotID)
		if !numericIDPattern.MatchString(depotID) {
			return nil, errors.New("enter a numeric depot ID for every depot")
		}
		if seen[depotID] {
			return nil, fmt.Errorf("depot %s is listed twice", depotID)
		}
		seen[depotID] = true
		id, _ := strconv.ParseUint(depotID, 10, 32)

		currentPath, err := requireReadableFile("the current manifest file for depot "+depotID, depot.CurrentManifestPath)
		if err != nil {
			return nil, err
		}
		targetPath, err := requireReadableFile("the previous version manifest for depot "+depotID, depot.TargetManifestPath)
		if err != nil {
			return nil, err
		}
		key, err := depotKeyFor(keys, depotID)
		if err != nil {
			return nil, err
		}
		current, err := loadManifest(currentPath, key)
		if err != nil {
			return nil, fmt.Errorf("the current manifest file for depot %s %w", depotID, err)
		}
		target, err := loadManifest(targetPath, key)
		if err != nil {
			return nil, fmt.Errorf("the previous version manifest for depot %s %w", depotID, err)
		}
		if err := checkManifestIdentity("the current manifest file", currentPath, current, id); err != nil {
			return nil, err
		}
		if err := checkManifestIdentity("the previous version manifest", targetPath, target, id); err != nil {
			return nil, err
		}
		if current.ManifestID == target.ManifestID {
			return nil, fmt.Errorf("depot %s: the current manifest file and the previous version manifest are the same version", depotID)
		}

		for _, file := range current.Files {
			rel, err := normalizeManifestPath(file.Name)
			if err != nil {
				return nil, fmt.Errorf("the current manifest file for depot %s has an %w", depotID, err)
			}
			if !file.IsDirectory() {
				currentFiles[pathKey(rel)] = plannedFile{rel: rel, size: file.Size}
			}
		}
		for _, file := range target.Files {
			rel, err := normalizeManifestPath(file.Name)
			if err != nil {
				return nil, fmt.Errorf("the previous version manifest for depot %s has an %w", depotID, err)
			}
			if !file.IsDirectory() {
				targetSet[pathKey(rel)] = true
				plan.targets = append(plan.targets, plannedFile{rel: rel, size: file.Size, linked: file.LinkTarget != ""})
			}
		}
		plan.depots = append(plan.depots, downgradeDepotPlan{depot: DowngradeDepot{DepotID: depotID, CurrentManifestPath: currentPath, TargetManifestPath: targetPath}, current: current, target: target})
		plan.summary.Depots = append(plan.summary.Depots, DowngradeDepotSummary{
			DepotID:           depotID,
			CurrentManifestID: strconv.FormatUint(current.ManifestID, 10),
			TargetManifestID:  strconv.FormatUint(target.ManifestID, 10),
			CurrentFiles:      countFiles(current),
			TargetFiles:       countFiles(target),
		})
	}

	state, others := inspectDepotState(resolvedRoot, seen)
	plan.summary.HadStateFolder = state
	plan.summary.OtherDepots = others
	plan.summary.Partial = len(others) > 0
	if plan.summary.Partial && !request.AcknowledgePartial {
		return plan, fmt.Errorf("this game folder also holds depot %s. Add its current and previous version manifest files so every depot is downgraded together, or confirm a partial downgrade", strings.Join(others, ", "))
	}

	if !plan.summary.Partial {
		for key, file := range currentFiles {
			if targetSet[key] {
				continue
			}
			full, err := resolveInside(resolvedRoot, file.rel)
			if err != nil {
				return nil, err
			}
			if stat, err := os.Lstat(full); err == nil && !stat.IsDir() {
				plan.removals = append(plan.removals, plannedFile{rel: file.rel, size: uint64(stat.Size())})
				plan.summary.RemoveBytes += stat.Size()
			}
		}
		sort.Slice(plan.removals, func(i, j int) bool { return plan.removals[i].rel < plan.removals[j].rel })
		plan.summary.RemoveCount = len(plan.removals)
	} else {
		plan.summary.Notes = append(plan.summary.Notes, "Partial downgrade: files are updated but nothing is removed, because the other depots might own them.")
	}
	if !state {
		plan.summary.Notes = append(plan.summary.Notes, "No .DepotDownloader folder was found. That is fine; one is created for you.")
	}
	return plan, nil
}

func countFiles(manifest *DepotManifest) int {
	count := 0
	for _, file := range manifest.Files {
		if !file.IsDirectory() {
			count++
		}
	}
	return count
}

// inspectDepotState reads .DepotDownloader/depot.config (when present) and lists installed depots that are not covered.
func inspectDepotState(root string, covered map[string]bool) (hasState bool, others []string) {
	config := filepath.Join(root, ".DepotDownloader", "depot.config")
	if stat, err := os.Stat(filepath.Dir(config)); err == nil && stat.IsDir() {
		hasState = true
	}
	installed, err := readDepotConfig(config)
	if err != nil {
		return hasState, nil
	}
	for depot := range installed {
		if id := strconv.FormatUint(uint64(depot), 10); !covered[id] {
			others = append(others, id)
		}
	}
	sort.Strings(others)
	return hasState, others
}

// readDepotConfig reads .DepotDownloader/depot.config: a deflate-compressed protobuf map of depot ID to the
// installed manifest ID. A missing file is an empty map.
func readDepotConfig(config string) (map[uint32]uint64, error) {
	file, err := os.Open(config)
	if os.IsNotExist(err) {
		return map[uint32]uint64{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(flate.NewReader(file), 8<<20))
	if err != nil {
		return nil, err
	}
	installed := map[uint32]uint64{}
	err = walkProto(data, func(field int, _ uint64, raw []byte) error {
		if field != 1 || raw == nil {
			return nil
		}
		var depot, manifest uint64
		_ = walkProto(raw, func(field int, value uint64, _ []byte) error {
			switch field {
			case 1:
				depot = value
			case 2:
				manifest = value
			}
			return nil
		})
		if depot != 0 {
			installed[uint32(depot)] = manifest
		}
		return nil
	})
	return installed, err
}

// writeDepotConfig writes the map back in the downloader's own format.
func writeDepotConfig(config string, installed map[uint32]uint64) error {
	var data []byte
	depots := make([]uint32, 0, len(installed))
	for depot := range installed {
		depots = append(depots, depot)
	}
	sort.Slice(depots, func(i, j int) bool { return depots[i] < depots[j] })
	for _, depot := range depots {
		entry := protoVarint(1, uint64(depot))
		entry = append(entry, protoVarint(2, installed[depot])...)
		data = append(data, protoBytes(1, entry)...)
	}
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		return err
	}
	file, err := os.Create(config)
	if err != nil {
		return err
	}
	compressor, err := flate.NewWriter(file, flate.DefaultCompression)
	if err != nil {
		file.Close()
		return err
	}
	if _, err := compressor.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := compressor.Close(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// downgradeArguments builds one downloader invocation per depot. Only Downgrade uses -sourcemanifestfile.
func downgradeArguments(plan *downgradePlan) [][]string {
	var commands [][]string
	for _, item := range plan.depots {
		commands = append(commands, []string{
			"-app", plan.request.AppID,
			"-depot", item.depot.DepotID,
			"-manifest", strconv.FormatUint(item.target.ManifestID, 10),
			"-manifestfile", item.depot.TargetManifestPath,
			"-sourcemanifestfile", item.depot.CurrentManifestPath,
			"-depotkeys", plan.request.KeysPath,
			"-dir", plan.root,
			"-validate",
		})
	}
	return commands
}
