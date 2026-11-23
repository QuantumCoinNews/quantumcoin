param(
    [Parameter(Mandatory)][ValidatePattern('^[0-9a-f]{40}$')][string]$Commit,
    [Parameter(Mandatory)][string]$EvidenceDirectory
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$lockPath = Join-Path $repo 'configs/windows-build-toolchain.json'
$lock = Get-Content -LiteralPath $lockPath -Raw | ConvertFrom-Json
$evidence = [IO.Path]::GetFullPath($EvidenceDirectory)
if ($evidence -eq $repo -or $evidence.StartsWith($repo + '\', [StringComparison]::OrdinalIgnoreCase) -or
    (Test-Path -LiteralPath $evidence)) { throw 'Choose a new evidence directory outside the checkout.' }

function Install-Archive($spec, [string]$root) {
    if ($spec.sha256 -cnotmatch '^[0-9a-f]{64}$' -or ([uri]$spec.url).Scheme -ne 'https') {
        throw 'Invalid toolchain lock entry.'
    }
    New-Item -ItemType Directory -Path $root | Out-Null
    $zip = Join-Path $root 'toolchain.zip'
    Invoke-WebRequest -Uri $spec.url -OutFile $zip
    if ((Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash -ine $spec.sha256) {
        throw 'Toolchain SHA-256 mismatch.'
    }
    Expand-Archive -LiteralPath $zip -DestinationPath (Join-Path $root 'unpacked')
}
New-Item -ItemType Directory -Path $evidence | Out-Null
@{ kind = 'unsignedCandidateCIGate'; sourceCommit = $Commit; ok = $false; phase = 'started' } |
    ConvertTo-Json | Set-Content -LiteralPath (Join-Path $evidence 'ci-gate.json') -Encoding utf8

if ((git -C $repo rev-parse HEAD) -ne $Commit) { throw 'Run the audit from the selected commit.' }
if ((git -C $repo status --porcelain)) { throw 'The audit requires a clean checkout.' }
& python (Join-Path $repo 'scripts/source_hygiene.py')
if ($LASTEXITCODE) { throw 'Source archive hygiene gate failed.' }

$stage = Join-Path ([IO.Path]::GetTempPath()) ('qc-toolchains-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $stage | Out-Null
Install-Archive $lock.go (Join-Path $stage 'go')
Install-Archive $lock.mingw (Join-Path $stage 'mingw')
Install-Archive $lock.ffmpeg (Join-Path $stage 'ffmpeg')
Install-Archive $lock.node (Join-Path $stage 'node')
$goBin = Join-Path $stage 'go/unpacked/go/bin'
$ccBin = Join-Path $stage 'mingw/unpacked/mingw64/bin'
$mediaBin = Join-Path $stage "ffmpeg/unpacked/ffmpeg-$($lock.ffmpeg.version)-essentials_build/bin"
$nodeBin = Join-Path $stage "node/unpacked/node-v$($lock.node.version)-win-x64"
if (!(Test-Path -LiteralPath (Join-Path $nodeBin 'node.exe') -PathType Leaf) -or
    !(Test-Path -LiteralPath (Join-Path $nodeBin 'node_modules/npm/bin/npm-cli.js') -PathType Leaf)) {
    throw 'Incomplete locked Node/npm toolchain.'
}
$env:Path = "$goBin;$ccBin;$mediaBin;$nodeBin;$env:Path"
$gcc = Join-Path $ccBin 'gcc.exe'
foreach ($name in @('gcc.exe', 'g++.exe', 'ar.exe', 'ld.exe')) {
    if (!(Test-Path -LiteralPath (Join-Path $ccBin $name) -PathType Leaf)) { throw 'Incomplete locked MinGW toolchain.' }
}
foreach ($name in @('ffmpeg.exe', 'ffprobe.exe')) {
    if (!(Test-Path -LiteralPath (Join-Path $mediaBin $name) -PathType Leaf)) { throw 'Incomplete locked FFmpeg toolchain.' }
}
$env:CC = $gcc
$env:CXX = Join-Path $ccBin 'g++.exe'
$env:AR = Join-Path $ccBin 'ar.exe'
$env:GOENV = 'off'
$env:GOWORK = 'off'
$env:GOTOOLCHAIN = 'local'

$candidate = Join-Path $stage 'candidate'
& python (Join-Path $repo 'scripts/verify_clean_stack.py') --output (Join-Path $evidence 'clean-stack.json')
if ($LASTEXITCODE) { throw 'Clean stack verification failed.' }
$stack = Get-Content -LiteralPath (Join-Path $evidence 'clean-stack.json') -Raw | ConvertFrom-Json
if ($stack.ok -isnot [bool] -or !$stack.ok -or $stack.head -cne $Commit -or
    $stack.workingTreeSnapshot -isnot [bool] -or $stack.workingTreeSnapshot) {
    throw 'The clean-stack receipt does not accept this clean commit.'
}
& python (Join-Path $repo 'scripts/verify_dependency_security.py') --output (Join-Path $evidence 'dependency-security.json')
if ($LASTEXITCODE) { throw 'Dependency security acceptance failed.' }
$security = Get-Content -LiteralPath (Join-Path $evidence 'dependency-security.json') -Raw | ConvertFrom-Json
if ($security.ok -isnot [bool] -or !$security.ok) { throw 'Dependency security receipt was not accepted.' }

$buildArguments = @('build', '--repo', $repo, '--commit', $Commit, '--version', 'ci-reproducibility',
    '--go-version', $lock.go.version, '--cc', $gcc, '--cc-version', $lock.mingw.version, '--output', $candidate)
& python (Join-Path $repo 'tools/qc_release.py') @buildArguments
if ($LASTEXITCODE) { throw 'Release reproduction failed.' }
$manifest = Get-Content -LiteralPath (Join-Path $candidate 'release-manifest.json') -Raw | ConvertFrom-Json
if ($manifest.sourceCommit -cne $Commit -or $manifest.goVersion -cne $lock.go.version) {
    throw 'The release candidate does not identify the tested source and toolchain.'
}

foreach ($name in @('release-manifest.json', 'provenance.json', 'sbom.cdx.json')) {
    Copy-Item -LiteralPath (Join-Path $candidate $name) -Destination $evidence
}
Copy-Item -LiteralPath $lockPath -Destination (Join-Path $evidence 'toolchain-lock.json')
$hashes = @{}
foreach ($name in @('clean-stack.json', 'dependency-security.json', 'release-manifest.json',
                    'provenance.json', 'sbom.cdx.json', 'toolchain-lock.json')) {
    $hashes[$name] = (Get-FileHash -LiteralPath (Join-Path $evidence $name) -Algorithm SHA256).Hash.ToLowerInvariant()
}
@{ kind = 'unsignedCandidateCIGate'; sourceCommit = $Commit; ok = $true;
   phase = 'candidate-verified'; signedReleaseAccepted = $false; evidenceSha256 = $hashes } |
    ConvertTo-Json | Set-Content -LiteralPath (Join-Path $evidence 'ci-gate.json') -Encoding utf8
Write-Output 'Two clean checkouts produced matching unsigned artifact bytes.'
