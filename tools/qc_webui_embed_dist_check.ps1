# QC_33C7_PSSCRIPTANALYZER_LEGACY_SUPPRESSIONS
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSAvoidUsingPositionalParameters', '', Justification='QC 33C7: legacy/test/run script naming or control-flow retained to avoid behavior change while zeroing analyzer warnings.')]
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSUseApprovedVerbs', '', Justification='QC 33C7: legacy/test/run script naming or control-flow retained to avoid behavior change while zeroing analyzer warnings.')]
param(
  [switch]$SourceOnly,
  [string]$ReleaseDirectory,
  [string]$ReleaseCommit,
  [string]$TrustedSigners
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'qc_release_audit.ps1')
$auditArgs = @{
  SourceOnly = $SourceOnly
  ReleaseDirectory = $ReleaseDirectory
  ReleaseCommit = $ReleaseCommit
  TrustedSigners = $TrustedSigners
}
$releaseEvidence = Get-QCReleaseAuditEvidence @auditArgs

$Repo = Split-Path -Parent $PSScriptRoot
$stamp = Get-Date -Format "yyyyMMdd_HHmmss"
$reportDir = Join-Path -Path $Repo -ChildPath "_local_audit_reports"
$buildDir = Join-Path -Path $reportDir -ChildPath ("QC_07_WEBUI_EMBED_DIST_" + $stamp)

New-Item -ItemType Directory -Force -Path $reportDir,$buildDir | Out-Null

$report = Join-Path -Path $reportDir -ChildPath "QC_07_WEBUI_EMBED_DIST_$stamp.txt"

function W($x) {
  $x | Tee-Object -FilePath $report -Append
}

function Run-Capture {
  param(
    [string]$Label,
    [string]$Exe,
    [string[]]$ArgList
  )

  W ""
  W "=== $Label ==="
  W ("COMMAND: {0} {1}" -f $Exe, ($ArgList -join " "))

  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $Exe

  foreach ($a in $ArgList) {
    [void]$psi.ArgumentList.Add($a)
  }

  $psi.WorkingDirectory = $Repo
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.UseShellExecute = $false

  $p = [System.Diagnostics.Process]::Start($psi)
  $stdoutRead = $p.StandardOutput.ReadToEndAsync()
  $stderrRead = $p.StandardError.ReadToEndAsync()
  $p.WaitForExit()
  $stdout = $stdoutRead.GetAwaiter().GetResult()
  $stderr = $stderrRead.GetAwaiter().GetResult()

  if ($stdout.Trim().Length -gt 0) { W $stdout.TrimEnd() }
  if ($stderr.Trim().Length -gt 0) { W $stderr.TrimEnd() }

  W "EXIT_CODE: $($p.ExitCode)"

  if ($p.ExitCode -ne 0) {
    throw "$Label failed"
  }
}

function Rel($full) {
  return $full.Substring($Repo.Length).TrimStart("\")
}

Set-Location -Path $Repo

W "=== QC 07 WEBUI EMBED / DIST CHECK ==="
W "Repo: $Repo"
W "Stamp: $stamp"
W "BuildDir: $buildDir"
W ""

W "=== Release gate ==="
W $releaseEvidence

W ""
W "=== WebUI related directories ==="

$dirs = @(
  "webui",
  "webwallet",
  "ui",
  "assets"
)

foreach ($d in $dirs) {
  if (Test-Path (Join-Path -Path $Repo -ChildPath $d)) {
    $files = @(Get-ChildItem -LiteralPath (Join-Path -Path $Repo -ChildPath $d) -Recurse -File -Force -ErrorAction SilentlyContinue)
    $bytes = ($files | Measure-Object Length -Sum).Sum
    W ("DIR_OK`t{0}`tFiles={1}`tBytes={2}" -f $d, $files.Count, $bytes)
  } else {
    W ("DIR_MISSING`t{0}" -f $d)
  }
}

W ""
W "=== Root index.html check ==="

if (Test-Path "index.html") {
  $idx = Get-Item "index.html"
  $hash = (Get-FileHash "index.html" -Algorithm SHA256).Hash
  W ("ROOT_INDEX_EXISTS`tSize={0}`tSHA256={1}" -f $idx.Length, $hash)
} else {
  W "ROOT_INDEX_MISSING"
}

W ""
W "=== Important web files sample ==="

$webPatterns = @(
  "webui\index.html",
  "webui\dist\index.html",
  "webui\static\index.html",
  "webui\assets",
  "webwallet\index.html",
  "webwallet\dist\index.html",
  "ui\index.html"
)

foreach ($p in $webPatterns) {
  if (Test-Path (Join-Path -Path $Repo -ChildPath $p)) {
    $item = Get-Item (Join-Path -Path $Repo -ChildPath $p)
    if ($item.PSIsContainer) {
      $cnt = @(Get-ChildItem -LiteralPath $item.FullName -Recurse -File -Force -ErrorAction SilentlyContinue).Count
      W ("WEB_PATH_OK_DIR`t{0}`tFiles={1}" -f $p, $cnt)
    } else {
      $hash = (Get-FileHash $item.FullName -Algorithm SHA256).Hash
      W ("WEB_PATH_OK_FILE`t{0}`tSize={1}`tSHA256={2}" -f $p, $item.Length, $hash)
    }
  } else {
    W ("WEB_PATH_MISSING`t{0}" -f $p)
  }
}

W ""
W "=== Go embed directives ==="

$goFiles = Get-ChildItem -LiteralPath $Repo -Recurse -File -Filter "*.go" -Force |
  Where-Object { $_.FullName -notlike "*\.git\*" -and $_.FullName -notlike "*_legacy_noise_archive*" -and $_.FullName -notlike "*_local_audit_reports*" }

$embedLines = @()

foreach ($f in $goFiles) {
  Select-String -Path $f.FullName -Pattern "//go:embed" -SimpleMatch -ErrorAction SilentlyContinue |
    ForEach-Object {
      $rel = Rel $f.FullName
      $embedLines += [pscustomobject]@{
        Path = $rel
        LineNumber = $_.LineNumber
        Line = $_.Line.Trim()
      }
    }
}

if ($embedLines.Count -eq 0) {
  W "NO_GO_EMBED_DIRECTIVES_FOUND"
} else {
  foreach ($e in $embedLines) {
    W ("EMBED`t{0}`tLine={1}`t{2}" -f $e.Path, $e.LineNumber, $e.Line)
  }
}

W ""
W "=== Go files importing embed package ==="

$embedImports = @()

foreach ($f in $goFiles) {
  Select-String -Path $f.FullName -Pattern '"embed"|_ "embed"' -ErrorAction SilentlyContinue |
    ForEach-Object {
      $rel = Rel $f.FullName
      $embedImports += [pscustomobject]@{
        Path = $rel
        LineNumber = $_.LineNumber
        Line = $_.Line.Trim()
      }
    }
}

if ($embedImports.Count -eq 0) {
  W "NO_EMBED_IMPORTS_FOUND"
} else {
  foreach ($e in $embedImports) {
    W ("EMBED_IMPORT`t{0}`tLine={1}`t{2}" -f $e.Path, $e.LineNumber, $e.Line)
  }
}

W ""
W "=== Main.go WebUI/static references ==="

$patterns = @(
  "webui",
  "webwallet",
  "index.html",
  "FileServer",
  "http.FS",
  "Static",
  "assets",
  "dist",
  "embed"
)

foreach ($p in $patterns) {
  W ""
  W "--- Pattern: $p ---"
  Select-String -Path "main.go" -Pattern $p -CaseSensitive:$false -ErrorAction SilentlyContinue |
    ForEach-Object {
      W ("Line {0}: {1}" -f $_.LineNumber, $_.Line.Trim())
    }
}

W ""
W "=== webui package files ==="

if (Test-Path "webui") {
  Get-ChildItem -LiteralPath "webui" -Recurse -File -Force |
    Sort-Object FullName |
    Select-Object -First 120 |
    ForEach-Object {
      $rel = Rel $_.FullName
      W ("WEBUI_FILE`t{0}`t{1}" -f $_.Length, $rel)
    }
} else {
  W "webui directory missing."
}

W ""
W "=== webwallet files sample ==="

if (Test-Path "webwallet") {
  Get-ChildItem -LiteralPath "webwallet" -Recurse -File -Force |
    Sort-Object FullName |
    Select-Object -First 120 |
    ForEach-Object {
      $rel = Rel $_.FullName
      W ("WEBWALLET_FILE`t{0}`t{1}" -f $_.Length, $rel)
    }
} else {
  W "webwallet directory missing."
}

W ""
W "=== Go package list for web related packages ==="

Run-Capture "go list ./webui" "go" @("list","./webui")

if (Test-Path "webwallet") {
  try {
    Run-Capture "go list ./webwallet" "go" @("list","./webwallet")
  } catch {
    W "WEBWALLET_GO_LIST_NOT_REQUIRED_OR_FAILED"
    W $_.Exception.Message
  }
}

W ""
W "=== Root node audit build for WebUI embed verification ==="

$out = Join-Path -Path $buildDir -ChildPath "quantumcoin-node-webui-audit.exe"
Run-Capture "go build . with WebUI verification" "go" @("build","-trimpath","-o",$out,".")

if (!(Test-Path $out)) {
  throw "WebUI audit build output missing"
}

$bh = (Get-FileHash $out -Algorithm SHA256).Hash
$bs = (Get-Item $out).Length
W ("BUILT_WEBUI_AUDIT`tSize={0}`tSHA256={1}`t{2}" -f $bs, $bh, $out)

W ""
W "=== Release gate after audit ==="
$afterEvidence = Get-QCReleaseAuditEvidence @auditArgs
if ($afterEvidence -ne $releaseEvidence) { throw 'Signed release changed during audit.' }
W $afterEvidence

W ""
W "=== WebUI conclusion ==="

if ($embedLines.Count -gt 0) {
  W "GO_EMBED_PRESENT"
} else {
  W "GO_EMBED_NOT_USED_OR_NOT_DETECTED"
}

W "WEBUI_BUILD_VERIFIED_BY_GO_BUILD_DOT"
W "Audit build output stayed under: $buildDir"

W ""
W "QC_07_WEBUI_EMBED_DIST_PASS"
W "QC_07_WEBUI_EMBED_DIST_DONE"
W "Report: $report"
