$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$work = Join-Path $here 'work'
$commit = 'c0f62fb7f020087f36ae76adfc51fde1446af344'

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
git clone https://github.com/SteamAutoCracks/DepotDownloaderMod.git $work
git -C $work checkout $commit
git -C $work apply --whitespace=nowarn (Join-Path $here 'downgrade.patch')

dotnet publish (Join-Path $work 'DepotDownloader/DepotDownloaderMod.csproj') `
  -c Release -r win-x64 --self-contained -p:PublishSingleFile=true -o (Join-Path $here 'out')

Write-Host "Built $(Join-Path $here 'out/DepotDownloaderMod.exe')"
