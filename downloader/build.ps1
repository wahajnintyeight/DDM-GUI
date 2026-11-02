$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$work = Join-Path $here 'work'
$commit = 'c0f62fb7f020087f36ae76adfc51fde1446af344'

# git and dotnet write progress to stderr, which Windows PowerShell 5.1 would treat as an error, so check exit codes instead.
function Invoke-Checked([string]$what, [scriptblock]$command) {
  $previous = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  & $command
  $code = $LASTEXITCODE
  $ErrorActionPreference = $previous
  if ($code -ne 0) { throw "$what failed (exit code $code)" }
}

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
Invoke-Checked 'git clone' { git clone -q https://github.com/SteamAutoCracks/DepotDownloaderMod.git $work }
Invoke-Checked 'git checkout' { git -C $work checkout -q $commit }
Invoke-Checked 'applying the downgrade patch' { git -C $work apply --whitespace=nowarn (Join-Path $here 'downgrade.patch') }

# Upstream pins SDK 9.0.100 in global.json. The project targets net9.0, which any newer SDK builds too.
Remove-Item (Join-Path $work 'global.json') -ErrorAction SilentlyContinue

Invoke-Checked 'dotnet publish' {
  dotnet publish (Join-Path $work 'DepotDownloader/DepotDownloaderMod.csproj') `
    -c Release -r win-x64 --self-contained -p:PublishSingleFile=true -o (Join-Path $here 'out')
}

Write-Host "Built $(Join-Path $here 'out/DepotDownloaderMod.exe')"
