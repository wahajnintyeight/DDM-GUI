# DepotDrop 0.2.7

- Download speed now comes from the selected DepotDownloaderMod process, so unrelated applications' network activity is excluded.
- Added a validation read-rate indicator. It counts logical checksum bytes and may include reads served from the operating system cache.
- Improved the speed graph and time-left estimate using completed chunk telemetry and smoothed progress samples.
- Updated pause, resume and cancel handling so output pipes held by child processes do not leave controls stuck.
- Added guidance for DepotDownloaderMod 3.4.0.3, which has buffered checksums, indexed manifest lookups and automatic HDD-aware validation concurrency. SSD/NVMe drives use the downloader's configured download concurrency.
- Updated the README with downloader setup, metric limitations and resume validation behavior.

## Requirements

- To see process-specific download and validation rates, select [DepotDownloaderMod 3.4.0.3](https://github.com/wahajnintyeight/DepotDownloaderMod/releases/download/DepotDownloaderMod_3.4.0.3/DepotDownloaderMod.exe). Older downloader builds can still run jobs, but do not emit the telemetry used by the live statistics.
- The DDM-GUI executable is a separate release asset. DepotDownloaderMod and depot keys are not bundled.
