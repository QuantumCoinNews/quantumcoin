param([Parameter(Mandatory=$true)][string]$OutputDirectory)
$ErrorActionPreference='Stop'
$repoRoot=Split-Path -Parent $PSScriptRoot
$checkDir=Join-Path ([IO.Path]::GetFullPath($OutputDirectory)) ('publisher-smoke-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path (Join-Path $checkDir 'dist') -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $repoRoot 'autopublisher/run-daily.ps1') -Destination (Join-Path $checkDir 'run-daily.ps1')
Set-Content -LiteralPath (Join-Path $checkDir 'publisher.mjs') -Value '// mock only' -Encoding utf8
Set-Content -LiteralPath (Join-Path $checkDir '.env') -Value 'QC_BOUNDARY_TEST_MARKER=must-not-load' -Encoding utf8
function global:node { $global:publisherBoundaryArgs=@($args); $global:LASTEXITCODE=0 }
function global:Invoke-RestMethod { throw 'Network publication forbidden in offline smoke check' }
try {
 & (Join-Path $checkDir 'run-daily.ps1') -Slot 2 -StateDir (Join-Path $checkDir 'state with spaces')
 if($LASTEXITCODE -ne 0){throw 'Wrapper failed'}
 if($env:QC_BOUNDARY_TEST_MARKER){throw 'Wrapper unexpectedly loaded dotenv'}
 $a=$global:publisherBoundaryArgs
 if($a.Count -ne 7 -or $a[0] -ne '--' -or $a[2] -ne 'run' -or $a[3] -ne '--slot' -or $a[4] -ne '2' -or $a[5] -ne '--state-dir' -or $a[6] -ne (Join-Path $checkDir 'state with spaces')){throw 'Typed argument forwarding failed'}
 if($a -contains '--publish'){throw 'Default wrapper enabled publication'}
 'PASS: canonical Node forwarding, explicit credentials only, dry-run default; no network.'
} finally {
 Remove-Item Function:node -ErrorAction SilentlyContinue
 Remove-Item Function:Invoke-RestMethod -ErrorAction SilentlyContinue
}
