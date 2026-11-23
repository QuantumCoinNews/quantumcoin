param([Parameter(Mandatory=$true)][string]$OutputDirectory)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$outputRoot = [IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Force -Path $outputRoot | Out-Null
$previousLocation = Get-Location
$previousEnvironment = @{}
foreach ($name in @('GOOS','GOARCH','CGO_ENABLED','GOTOOLCHAIN')) { $previousEnvironment[$name] = [Environment]::GetEnvironmentVariable($name,'Process') }
$results = @()
try {
    Set-Location $repoRoot
    $env:CGO_ENABLED = '0'
    $env:GOTOOLCHAIN = 'local'
    $toolchain = (& go version | Out-String).Trim()
    $deps = @(& go list -deps .)
    if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect headless dependencies' }
    if ($deps | Where-Object { $_ -match '^fyne.io/' }) { throw 'Headless root unexpectedly imports GUI packages' }
    foreach ($target in @('windows/amd64','linux/amd64','linux/arm64','darwin/amd64','darwin/arm64')) {
        $env:GOOS,$env:GOARCH = $target.Split('/')
        $name = 'quantumcoin-' + $target.Replace('/','-')
        if ($env:GOOS -eq 'windows') { $name += '.exe' }
        $binary = Join-Path $outputRoot $name
        & go build -mod=readonly -o $binary . *> (Join-Path $outputRoot ($name + '.log'))
        $code = $LASTEXITCODE
        $hash = if ($code -eq 0) { (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant() } else { $null }
        $results += [pscustomobject]@{target=$target; exitCode=$code; sha256=$hash}
    }
    [pscustomobject]@{toolchain=$toolchain; cgoEnabled=$false; guiImported=$false; verification='cross-build only; foreign binaries not executed'; results=$results} | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $outputRoot 'matrix.json') -Encoding utf8
    $results | Format-Table -AutoSize
    if ($results | Where-Object {$_.exitCode -ne 0}) { throw 'One or more headless cross-builds failed' }
} finally {
    Set-Location $previousLocation
    foreach ($name in $previousEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name,$previousEnvironment[$name],'Process') }
}
