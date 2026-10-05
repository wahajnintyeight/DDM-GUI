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

var (
	manifestNamePattern = regexp.MustCompile(`^(\d+)_(\d+)\.manifest$`)
	numericIDPattern    = regexp.MustCompile(`^\d+$`)
	progressPattern     = regexp.MustCompile(`(?:^|\s)(\d{1,3}(?:\.\d+)?)%`)
	totalBytesPattern   = regexp.MustCompile(`(?i)^Total downloaded:\s*([\d,]+)\s+bytes(?:\s+\(([\d,]+)\s+bytes\s+uncompressed\))?`)
	ansiPattern         = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
)

type ManifestInfo struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	DepotID    string `json:"depotId"`
	ManifestID string `json:"manifestId"`
}

type DownloadRequest struct {
	ManifestPath   string `json:"manifestPath"`
	GameFolder     string `json:"gameFolder"`
	DownloaderPath string `json:"downloaderPath"`
	KeysPath       string `json:"keysPath"`
	AppID          string `json:"appId"`
	DepotID        string `json:"depotId"`
	Validate       bool   `json:"validate"`
}

type DownloadUpdate struct {
	State             string  `json:"state"`
	Progress          float64 `json:"progress"`
	Message           string  `json:"message"`
	Line              string  `json:"line,omitempty"`
	DownloadedBytes   int64   `json:"downloadedBytes"`
	UncompressedBytes int64   `json:"uncompressedBytes"`
	ElapsedSeconds    float64 `json:"elapsedSeconds"`
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
		return ManifestInfo{}, errors.New("manifest filename must look like DEPOTID_MANIFESTID.manifest")
	}

	return ManifestInfo{Path: path, Name: name, DepotID: parts[1], ManifestID: parts[2]}, nil
}

func downloadCommand(info ManifestInfo, gameFolder, downloaderPath, keysPath, appID, depotID string, validate bool) (string, []string, error) {
	gameFolder = filepath.Clean(strings.TrimSpace(gameFolder))
	if gameFolder == "." || gameFolder == "" {
		return "", nil, errors.New("choose a game folder")
	}
	if stat, err := os.Stat(gameFolder); err != nil || !stat.IsDir() {
		return "", nil, errors.New("the game folder does not exist; choose an existing folder")
	}

	appID = strings.TrimSpace(appID)
	if !numericIDPattern.MatchString(appID) {
		return "", nil, errors.New("enter a numeric game app ID")
	}
	depotID = strings.TrimSpace(depotID)
	if depotID == "" {
		depotID = info.DepotID
	}
	if !numericIDPattern.MatchString(depotID) {
		return "", nil, errors.New("enter a numeric depot ID or choose a matching manifest")
	}

	executable := filepath.Clean(strings.TrimSpace(downloaderPath))
	keys := filepath.Clean(strings.TrimSpace(keysPath))
	for label, path := range map[string]string{"DepotDownloaderMod.exe": executable, "keys.txt": keys} {
		if path == "." || path == "" {
			return "", nil, fmt.Errorf("choose %s", label)
		}
		if stat, err := os.Stat(path); err != nil || stat.IsDir() {
			return "", nil, fmt.Errorf("%s could not be opened at the selected path", label)
		}
	}

	args := []string{
		"-app", appID,
		"-depot", depotID,
		"-manifest", info.ManifestID,
		"-manifestfile", info.Path,
		"-depotkeys", keys,
		"-dir", gameFolder,
	}
	if validate {
		args = append(args, "-validate")
	}
	return executable, args, nil
}

func (a *App) StartDownload(request DownloadRequest) error {
	info, err := inspectManifest(request.ManifestPath)
	if err != nil {
		return err
	}
	executable, args, err := downloadCommand(info, request.GameFolder, request.DownloaderPath, request.KeysPath, request.AppID, request.DepotID, request.Validate)
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
	a.paused = false
	a.resume = make(chan struct{}, 1)
	attemptCtx, pauseCancel := context.WithCancelCause(ctx)
	a.pauseCancel = pauseCancel
	a.mu.Unlock()

	go a.runDownload(ctx, attemptCtx, executable, args, filepath.Dir(executable), func(update DownloadUpdate) {
		runtime.EventsEmit(a.ctx, "download:update", update)
	})
	return nil
}

func (a *App) CancelDownload() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

var errDownloadPaused = errors.New("download paused")

func (a *App) PauseDownload() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.running || a.paused || a.pauseCancel == nil {
		return errors.New("no active download to pause")
	}
	a.pauseCancel(errDownloadPaused)
	return nil
}

func (a *App) ResumeDownload() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.running || !a.paused {
		return errors.New("no paused download to resume")
	}
	a.paused = false
	a.resume <- struct{}{}
	return nil
}

func (a *App) runDownload(ctx, attemptCtx context.Context, executable string, args []string, workingDirectory string, emit func(DownloadUpdate)) {
	var update DownloadUpdate
	activeSeconds := 0.0
	for {
		update = downloadAttempt(attemptCtx, executable, args, workingDirectory, emit)
		activeSeconds += update.ElapsedSeconds
		update.ElapsedSeconds = activeSeconds
		a.mu.Lock()
		a.pauseCancel(nil)
		a.pauseCancel = nil
		if ctx.Err() != nil || update.State != "paused" {
			if ctx.Err() != nil {
				update.State, update.Message = "cancelled", "Download cancelled"
			}
			if a.cancel != nil {
				a.cancel()
			}
			a.running, a.paused, a.cancel = false, false, nil
			emit(update)
			a.mu.Unlock()
			return
		}
		a.paused = true
		emit(update)
		a.mu.Unlock()
		select {
		case <-ctx.Done():
		case <-a.resume:
		}
		a.mu.Lock()
		if ctx.Err() != nil {
			update.State, update.Message = "cancelled", "Download cancelled"
			if a.cancel != nil {
				a.cancel()
			}
			a.running, a.paused, a.cancel = false, false, nil
			emit(update)
			a.mu.Unlock()
			return
		}
		attemptCtx, a.pauseCancel = context.WithCancelCause(ctx)
		a.paused = false
		a.mu.Unlock()
	}
}

func downloadAttempt(ctx context.Context, executable string, args []string, workingDirectory string, emit func(DownloadUpdate)) DownloadUpdate {
	if ctx.Err() != nil {
		if errors.Is(context.Cause(ctx), errDownloadPaused) {
			return DownloadUpdate{State: "paused", Message: "Download paused. Resume to continue downloading."}
		}
		return DownloadUpdate{State: "cancelled", Message: "Download cancelled"}
	}
	emit(DownloadUpdate{State: "running", Message: "Connecting to Steam…"})

	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = workingDirectory
	stdout, err := command.StdoutPipe()
	if err != nil {
		return DownloadUpdate{State: "error", Message: "Could not read downloader output: " + err.Error()}
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return DownloadUpdate{State: "error", Message: "Could not read downloader errors: " + err.Error()}
	}
	startedAt := time.Now()
	if err := command.Start(); err != nil {
		state, message := "error", "DepotDownloaderMod could not start: "+err.Error()
		if errors.Is(context.Cause(ctx), errDownloadPaused) {
			state, message = "paused", "Download paused. Resume to continue downloading."
		}
		return DownloadUpdate{State: state, Message: message}
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
	downloadedBytes := int64(0)
	uncompressedBytes := int64(0)
	lastProgressEmit := time.Time{}
	for line := range lines {
		line = strings.TrimSpace(ansiPattern.ReplaceAllString(line, ""))
		if line == "" {
			continue
		}
		if downloaded, uncompressed, ok := parseDownloadTotals(line); ok {
			downloadedBytes = downloaded
			uncompressedBytes = uncompressed
		}
		if parsed, ok := parseProgress(line); ok {
			progress = parsed
			if time.Since(lastProgressEmit) < 75*time.Millisecond {
				continue
			}
			lastProgressEmit = time.Now()
		}
		emit(DownloadUpdate{State: "running", Progress: progress, Message: "Downloading files", Line: line, DownloadedBytes: downloadedBytes, UncompressedBytes: uncompressedBytes})
	}

	waitErr := command.Wait()
	elapsedSeconds := time.Since(startedAt).Seconds()
	if ctx.Err() != nil {
		state, message := "cancelled", "Download cancelled"
		if errors.Is(context.Cause(ctx), errDownloadPaused) {
			state, message = "paused", "Download paused. Resume to continue downloading."
		}
		return DownloadUpdate{State: state, Progress: progress, Message: message, DownloadedBytes: downloadedBytes, UncompressedBytes: uncompressedBytes, ElapsedSeconds: elapsedSeconds}
	}
	if waitErr != nil {
		return DownloadUpdate{State: "error", Progress: progress, Message: "DepotDownloaderMod stopped: " + waitErr.Error(), DownloadedBytes: downloadedBytes, UncompressedBytes: uncompressedBytes, ElapsedSeconds: elapsedSeconds}
	}
	return DownloadUpdate{State: "complete", Progress: 100, Message: "Download complete", DownloadedBytes: downloadedBytes, UncompressedBytes: uncompressedBytes, ElapsedSeconds: elapsedSeconds}
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

func parseDownloadTotals(line string) (int64, int64, bool) {
	match := totalBytesPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 3 {
		return 0, 0, false
	}

	parseBytes := func(value string) (int64, error) {
		return strconv.ParseInt(strings.ReplaceAll(value, ",", ""), 10, 64)
	}
	downloaded, err := parseBytes(match[1])
	if err != nil {
		return 0, 0, false
	}
	uncompressed := int64(0)
	if match[2] != "" {
		uncompressed, err = parseBytes(match[2])
		if err != nil {
			return 0, 0, false
		}
	}
	return downloaded, uncompressed, true
}
