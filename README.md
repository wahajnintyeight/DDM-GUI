# DepotDrop

> A small Windows desktop UI for downloading and validating a Steam depot without typing a long DepotDownloader command.

DepotDrop wraps the local `DepotDownloaderMod.exe` workflow with guided setup: choose the manifest, app ID, downloader executable, depot keys, and destination folder, then watch the real downloader progress in the app.

## Built with

[![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-DF5D36?style=for-the-badge&logo=wails&logoColor=white)](https://github.com/wailsapp/wails)
[![JavaScript](https://img.shields.io/badge/JavaScript-F7DF1E?style=for-the-badge&logo=javascript&logoColor=black)](https://developer.mozilla.org/docs/Web/JavaScript)
[![Vite](https://img.shields.io/badge/Vite-646CFF?style=for-the-badge&logo=vite&logoColor=white)](https://vite.dev/)
[![Web Awesome](https://img.shields.io/badge/Web%20Awesome-4774E7?style=for-the-badge)](https://webawesome.com/)
[![Steam Store](https://img.shields.io/badge/Steam%20Store%20Search-000000?style=for-the-badge&logo=steam&logoColor=white)](https://store.steampowered.com/)
[![DepotDownloaderMod](https://img.shields.io/badge/DepotDownloaderMod-181717?style=for-the-badge&logo=github&logoColor=white)](https://github.com/SteamAutoCracks/DepotDownloaderMod)
[![Playwright](https://img.shields.io/badge/Playwright-2EAD33?style=for-the-badge&logo=playwright&logoColor=white)](https://playwright.dev/)

![DepotDrop desktop interface](screenshots/depotdrop-overview.png)

## Download for Windows

Download the latest ready-to-run `.exe` from the [GitHub Releases page](https://github.com/wahajnintyeight/DDM-GUI/releases/latest). The packaged app requires the Windows WebView2 Runtime, which is already present on most current Windows installations.

## What it does

- Reads any manifest filename matching `DEPOTID_MANIFESTID.manifest`.
- Extracts the depot ID and manifest ID from that filename; the values are not hardcoded to a particular game.
- Searches Steam by name after one character, shows up to 10 app results, and fills the app ID on selection.
- Accepts any numeric game app ID, so the UI is not tied to one community or game.
- Lets users choose `DepotDownloaderMod.exe` and `keys.txt` from any local paths.
- Offers a **Validate** checkbox (on by default) to include or omit `-validate`, using process arguments instead of a shell command.
- Streams percentage progress, the current file, and a bounded output log.
- Shows the final downloaded size, uncompressed size, and elapsed time from the downloader output.
- Supports pause/resume, cancellation, and clear download states.

## Quick start

You need these local inputs:

```text
DepotDownloaderMod.exe
keys.txt
<depotID>_<manifestID>.manifest
```

### `keys.txt` format

Use plain text with one depot key per line:

```text
<depotID>;<hexKey>
```

For example, a file containing keys for two depots has two lines. Use the 64-character hexadecimal depot decryption key supplied for each depot; do not include quotes or extra formatting.

They do not need to be in the same folder. DepotDrop never downloads, embeds, uploads, or commits the downloader or keys file; users select their own trusted copies.

Then:

1. Build or download `DDM-GUI.exe`.
2. Open the app.
3. Search for a Steam game and select a result, or choose **Enter app ID** to type it manually. Search uses the public Steam Store endpoint without an API key; results can include DLC and soundtracks.
4. Choose a `.manifest` file named `DEPOTID_MANIFESTID.manifest`. The depot ID is filled from its filename when possible.
5. Choose `DepotDownloaderMod.exe`.
6. Choose `keys.txt`.
7. Choose an existing game destination folder.
8. Leave **Validate** checked to verify all existing files, or uncheck it to use the downloader's normal checks.
9. Select **Start download**.

**Pause** stops the downloader and keeps the downloaded files. **Resume** restarts the same command with the selected validation setting. With **Validate** enabled, all existing files are checked; when disabled, the downloader can still check files needed for recovery or updates. Validation can take time and interrupted or unwritten chunks may download again. Keep the destination folder and its `.DepotDownloader` metadata. Paused setup stays locked until you resume or cancel; after closing the app, select the same inputs and destination and start again to recover existing chunks. Elapsed time excludes time spent paused; downloaded byte totals come from the latest downloader run.

The app does not read or transmit the contents of the manifest, depot keys, or downloader executable. It passes their selected local paths directly to the downloader.

## Build from source

Requirements:

- Go 1.23+
- Node.js 20+
- Wails CLI 2.11+
- WebView2 Runtime on Windows

```powershell
npm install --prefix frontend
go test ./...
wails build -clean
```

The production executable is written to `build/bin/DDM-GUI.exe`.

For live development:

```powershell
wails dev
```

## Project layout

| Path | Purpose |
| --- | --- |
| `app.go` | Wails lifecycle and native file/folder dialogs |
| `download.go` | Manifest parsing, command construction, process execution, and progress parsing |
| `download_test.go` | Parser and command-contract checks |
| `frontend/src/main.js` | Guided setup and live download state |
| `frontend/src/style.css` | Responsive Cobalt/Workbench interface |
| `frontend/tokens.css` | Shared visual tokens |

## Safety notes

DepotDrop validates the selected paths, requires the app ID and keys file, and accepts a blank depot ID only when the manifest filename provides one. It passes arguments directly to the executable, which avoids shell quoting problems for folders containing spaces. Review the downloader and keys bundle and their provenance before using them.

## License

No license has been selected yet. Add one before redistributing the project.
