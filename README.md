# DepotDrop

> A small Windows desktop UI for downloading and validating a Steam depot without typing a long DepotDownloader command.

DepotDrop wraps the supplied `DepotDownloaderMod.exe` workflow in two choices: select the manifest, select the destination folder, then watch the real downloader progress in the app.

## What it does

- Reads `3468651_795493147105120679.manifest` filenames safely.
- Uses `3468651` as the depot ID and `795493147105120679` as the manifest ID.
- Finds `DepotDownloaderMod.exe` and `keys.txt` beside the selected manifest.
- Runs the downloader with `-validate` using process arguments instead of a shell command.
- Streams percentage progress, the current file, and a bounded output log.
- Supports cancellation and clear startup, error, cancelled, and completed states.

## Quick start

Keep these files together in the downloader bundle:

```text
DepotDownloaderMod.exe
keys.txt
3468651_795493147105120679.manifest
```

Then:

1. Build or download `DDM-GUI.exe`.
2. Open the app.
3. Choose the `.manifest` file.
4. Choose an existing game destination folder.
5. Select **Start download**.

The app does not embed, upload, or commit manifests, depot keys, or the downloader executable. They remain local beside the manifest.

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
| `frontend/src/main.js` | Two-path workflow and live download state |
| `frontend/src/style.css` | Responsive Cobalt/Workbench interface |
| `frontend/tokens.css` | Shared visual tokens |

## Safety notes

DepotDrop validates the selected paths and requires the downloader and keys file to be beside the manifest. It passes arguments directly to the executable, which avoids shell quoting problems for folders containing spaces. Review the downloader bundle and its provenance before using it.

## License

No license has been selected yet. Add one before redistributing the project.
