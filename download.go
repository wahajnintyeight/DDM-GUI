package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const appID = "3468650"

var (
	manifestNamePattern = regexp.MustCompile(`^(\d+)_(\d+)\.manifest$`)
	progressPattern     = regexp.MustCompile(`(?:^|\s)(\d{1,3}(?:\.\d+)?)%`)
	ansiPattern         = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
)

type ManifestInfo struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	DepotID    string `json:"depotId"`
	ManifestID string `json:"manifestId"`
}

type DownloadRequest struct {
	ManifestPath string `json:"manifestPath"`
	GameFolder   string `json:"gameFolder"`
}

type DownloadUpdate struct {
	State    string  `json:"state"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
	Line     string  `json:"line,omitempty"`
}

func inspectManifest(path string) (ManifestInfo, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	stat, err := os.Stat(path)
	if err != nil {
		return ManifestInfo{}, fmt.Errorf("manifest could not be opened: %w", err)
	}
	if stat.IsDir() {
		return ManifestInfo{}, errors.New("the selected manifest is a folder; choose a .manifest file")
	}

	name := filepath.Base(path)
	parts := manifestNamePattern.FindStringSubmatch(name)
	if len(parts) != 3 {
		return ManifestInfo{}, errors.New("manifest filename must look like 3468651_795493147105120679.manifest")
	}

	return ManifestInfo{Path: path, Name: name, DepotID: parts[1], ManifestID: parts[2]}, nil
}

func downloadCommand(info ManifestInfo, gameFolder string) (string, []string, error) {
	gameFolder = filepath.Clean(strings.TrimSpace(gameFolder))
	if gameFolder == "." || gameFolder == "" {
		return "", nil, errors.New("choose a game folder")
	}
	if stat, err := os.Stat(gameFolder); err != nil || !stat.IsDir() {
		return "", nil, errors.New("the game folder does not exist; choose an existing folder")
	}

	bundleFolder := filepath.Dir(info.Path)
	executable := filepath.Join(bundleFolder, "DepotDownloaderMod.exe")
	keys := filepath.Join(bundleFolder, "keys.txt")
	for label, path := range map[string]string{"DepotDownloaderMod.exe": executable, "keys.txt": keys} {
		if stat, err := os.Stat(path); err != nil || stat.IsDir() {
			return "", nil, fmt.Errorf("%s must be beside the selected manifest", label)
		}
	}

	args := []string{
		"-app", appID,
		"-depot", info.DepotID,
		"-manifest", info.ManifestID,
		"-manifestfile", info.Path,
		"-depotkeys", keys,
		"-dir", gameFolder,
		"-validate",
	}
	return executable, args, nil
}

func (a *App) StartDownload(request DownloadRequest) error {
	info, err := inspectManifest(request.ManifestPath)
	if err != nil {
		return err
	}
	executable, args, err := downloadCommand(info, request.GameFolder)
	if err != nil {
		return err
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("a download is already running")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	a.running = true
	a.mu.Unlock()

	go a.runDownload(ctx, executable, args, filepath.Dir(info.Path))
	return nil
}

func (a *App) CancelDownload() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) runDownload(ctx context.Context, executable string, args []string, workingDirectory string) {
	defer func() {
		a.mu.Lock()
		a.running = false
		a.cancel = nil
		a.mu.Unlock()
	}()

	emit := func(update DownloadUpdate) { runtime.EventsEmit(a.ctx, "download:update", update) }
	emit(DownloadUpdate{State: "running", Message: "Connecting to Steam…"})

	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = workingDirectory
	stdout, err := command.StdoutPipe()
	if err != nil {
		emit(DownloadUpdate{State: "error", Message: "Could not read downloader output: " + err.Error()})
		return
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		emit(DownloadUpdate{State: "error", Message: "Could not read downloader errors: " + err.Error()})
		return
	}
	if err := command.Start(); err != nil {
		emit(DownloadUpdate{State: "error", Message: "DepotDownloaderMod could not start: " + err.Error()})
		return
	}

	lines := make(chan string, 64)
	var readers sync.WaitGroup
	readers.Add(2)
	go scanLines(stdout, lines, &readers)
	go scanLines(stderr, lines, &readers)
	go func() {
		readers.Wait()
		close(lines)
	}()

	progress := 0.0
	lastProgressEmit := time.Time{}
	for line := range lines {
		line = strings.TrimSpace(ansiPattern.ReplaceAllString(line, ""))
		if line == "" {
			continue
		}
		if parsed, ok := parseProgress(line); ok {
			progress = parsed
			if time.Since(lastProgressEmit) < 75*time.Millisecond {
				continue
			}
			lastProgressEmit = time.Now()
		}
		emit(DownloadUpdate{State: "running", Progress: progress, Message: "Downloading and validating files", Line: line})
	}

	waitErr := command.Wait()
	if ctx.Err() != nil {
		emit(DownloadUpdate{State: "cancelled", Progress: progress, Message: "Download cancelled"})
		return
	}
	if waitErr != nil {
		emit(DownloadUpdate{State: "error", Progress: progress, Message: "DepotDownloaderMod stopped: " + waitErr.Error()})
		return
	}
	emit(DownloadUpdate{State: "complete", Progress: 100, Message: "Download complete and validated"})
}

func scanLines(reader io.Reader, lines chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Split(splitOutputLines)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
}

func splitOutputLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for index, char := range data {
		if char == '\n' || char == '\r' {
			advance = index + 1
			if char == '\r' && len(data) > advance && data[advance] == '\n' {
				advance++
			}
			return advance, data[:index], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func parseProgress(line string) (float64, bool) {
	match := progressPattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return 0, false
	}
	progress, err := strconv.ParseFloat(match[1], 64)
	if err != nil || progress < 0 || progress > 100 {
		return 0, false
	}
	return progress, true
}
