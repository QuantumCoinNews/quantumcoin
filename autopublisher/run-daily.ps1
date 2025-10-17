Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'

$root='C:\Projects\quantumcoin\autopublisher'
Set-Location $root

# Node PATH (Task Scheduler PATH boş olabiliyor)
$nodeDir='C:\Program Files\nodejs'
if (Test-Path $nodeDir){ $env:Path="$nodeDir;$env:Path" }

# Basit log
$logDir = Join-Path $root 'logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$log = Join-Path $logDir ("sched-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".log")
function LOG([string]$m){ ("[{0}] {1}" -f (Get-Date -Format "HH:mm:ss"),$m) | Tee-Object -FilePath $log -Append | Out-Null }

# .env yükle
Get-Content .\.env | ? {$_ -match '^\s*[^#]'} | % { $k,$v = $_ -split '=',2; [Environment]::SetEnvironmentVariable($k,$v,'Process') }
LOG "ENV loaded"

# OUT klasörü garanti
New-Item -ItemType Directory -Force -Path .\src\content\out | Out-Null
$lastJson=".\src\content\out\last.json"

# İçeriği üret (child process env devralır; --env-file'a gerek yok)
LOG "gen-content start"
$nodeOut = & node .\scripts\gen-content.mjs 2>&1
$exit=$LASTEXITCODE
$nodeOut | Set-Content -Path $lastJson -Encoding UTF8
if($exit -ne 0){ LOG ("gen-content ERR: " + ($nodeOut -join ' ' -replace '\s+',' ')); exit 2 }

try { $p = (Get-Content $lastJson -Raw) | ConvertFrom-Json } catch { LOG ("json parse ERR: " + $_.Exception.Message); exit 2 }
LOG "gen-content OK"

# Telegram
try {
  & pwsh -NoProfile -ExecutionPolicy Bypass -File .\publish-all.ps1 -Text $p.telegram -Platforms telegram
  LOG "telegram exit=$LASTEXITCODE"
} catch { LOG ("telegram ERR: " + $_.Exception.Message); exit 2 }

Start-Sleep -Seconds 2

# X (test/ek yok)
$xt = $p.x
try {
  & pwsh -NoProfile -ExecutionPolicy Bypass -File .\publish-all.ps1 -Text $xt -Platforms x
  LOG "x exit=$LASTEXITCODE"
} catch { LOG ("x ERR: " + $_.Exception.Message); exit 2 }

LOG "DONE"
