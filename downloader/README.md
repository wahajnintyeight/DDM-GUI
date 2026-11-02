# DepotDownloaderMod with downgrade support

DepotDrop's **Downgrade** feature needs a DepotDownloaderMod build that understands one extra, opt-in option:

```text
-sourcemanifestfile <file>   manifest of the version installed now (the old manifest)
-manifestfile <file>         manifest of the older version you want (the new/target manifest)
```

Normal downloads never use it. `downgrade.patch` changes behaviour **only** when `-sourcemanifestfile` is present:

- the supplied source manifest is the old manifest and `-manifestfile` is the target; neither comes from `.DepotDownloader`
- validation is always on, matching chunks are reused, only missing or mismatched target chunks are downloaded
- every target file ends at exactly its manifest size, even when all of its chunks already match
- a file that already matches the target (for example from an interrupted run) is not downloaded again
- the downloader's own stale-file deletion is skipped; DepotDrop removes old files itself, and only after every file succeeded
- `.DepotDownloader/depot.config` is created or updated as usual, and records the target manifest on success

The patch applies to upstream `SteamAutoCracks/DepotDownloaderMod` at commit `c0f62fb7f020087f36ae76adfc51fde1446af344`.

## Build

You need git and a .NET SDK, version 9 or newer (the script ignores upstream's SDK 9.0.100 pin so a newer SDK such as 10 works).

```powershell
./build.ps1            # clones upstream, applies the patch, publishes a single-file DepotDownloaderMod.exe
```

The result is `downloader/out/DepotDownloaderMod.exe`. Choose it in DepotDrop's Downgrade tab (or anywhere you pick the downloader). DepotDrop checks that the chosen exe lists `-sourcemanifestfile` before it starts a downgrade, so an unpatched build can never compare the target manifest with itself.
