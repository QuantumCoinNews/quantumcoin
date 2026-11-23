param(
  [string]$ApiBase = $env:QC_API_BASE,
  [string]$Address = $env:ADDR,
  [string]$ReleaseDir = $PSScriptRoot,
  [ValidateRange(1, 600000)]
  [int]$DelayMs = 5000
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($ApiBase)) { $ApiBase = 'http://127.0.0.1:28082' }
$uri = $null
if (-not [Uri]::TryCreate($ApiBase, [UriKind]::Absolute, [ref]$uri) -or
    $uri.Scheme -notin @('http', 'https') -or $uri.UserInfo -or $uri.Query -or $uri.Fragment -or
    $uri.AbsolutePath -notmatch '^/*$' -or (-not $uri.IsLoopback -and $uri.Scheme -ne 'https')) {
  throw 'ApiBase must be a loopback HTTP(S) or remote HTTPS origin without credentials or a path.'
}
$base = $uri.GetLeftPart([UriPartial]::Authority)
if ([string]::IsNullOrWhiteSpace($env:QC_API_TOKEN)) {
  throw 'Set QC_API_TOKEN in the process environment before starting the miner client.'
}
$headers = @{ 'X-QC-API-KEY' = $env:QC_API_TOKEN.Trim() }

if ([string]::IsNullOrWhiteSpace($Address)) {
  $Address = $env:QC_ADDR
}
if ([string]::IsNullOrWhiteSpace($Address)) {
  foreach ($name in @('miner_address.txt', 'wallet_address.txt')) {
    $path = Join-Path $ReleaseDir $name
    if (Test-Path -LiteralPath $path -PathType Leaf) {
      $Address = (Get-Content -LiteralPath $path -Raw).Trim()
      break
    }
  }
}
$Address = ([string]$Address).Trim()
if ($Address -cnotmatch '^[1-9A-HJ-NP-Za-km-z]{26,64}$') {
  throw 'Set a valid Base58 reward address before starting the miner client.'
}

$stopPath = Join-Path $ReleaseDir 'miner_stop.flag'
$logPath = Join-Path $ReleaseDir 'miner_out.log'
function Write-MinerLog([string]$Message) {
  $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') $Message"
  Add-Content -LiteralPath $logPath -Value $line -Encoding utf8
  Write-Host $line
}

function Invoke-MinerRequest([string]$Path, [string]$Method, [int]$Timeout, [string]$Body = '') {
  $request = @{
    Uri = $base + $Path
    Method = $Method
    Headers = $headers
    TimeoutSec = $Timeout
    MaximumRedirection = 0
    SkipHttpErrorCheck = $true
    NoProxy = $true
  }
  if ($Body) {
    $request['ContentType'] = 'application/json'
    $request['Body'] = $Body
  }
  try {
    $response = Invoke-WebRequest @request
  } catch {
    throw 'Node request failed. Check the running node before restarting the miner client.'
  }
  if ($response.StatusCode -eq 401) {
    throw 'Node rejected the API token (HTTP 401).'
  }
  if ($response.StatusCode -eq 403) {
    throw 'Node denied this operation (HTTP 403). Mining requires a writable node.'
  }
  if ($response.StatusCode -eq 409) {
    throw 'Node rejected the expected isolated local genesis (HTTP 409).'
  }
  if ($response.StatusCode -ne 200) {
    throw "Node returned HTTP $([int]$response.StatusCode)."
  }
  try {
    return ConvertFrom-Json -InputObject $response.Content -AsHashtable
  } catch {
    throw 'Node returned invalid JSON.'
  }
}

if (Test-Path -LiteralPath $stopPath -PathType Leaf) {
  Remove-Item -LiteralPath $stopPath
}
$body = @{ address = $Address } | ConvertTo-Json -Compress
try {
  $health = Invoke-MinerRequest '/api/health' 'Get' 5
  if ($health.ok -isnot [bool] -or -not $health.ok) { throw 'Node health response did not confirm readiness.' }
  if ($health.apiReadOnly -isnot [bool] -or $health.apiReadOnly) {
    throw 'The full-block miner requires a node that explicitly reports writable mode.'
  }
  $identity = $health.networkIdentity
  if ($identity.version -cne 'QC_CHAIN_NETWORK_IDENTITY_V1' -or $identity.network -cne 'local-release' -or
      $identity.genesisDomain -cne 'QC_LOCAL_RELEASE_GENESIS_V2' -or $identity.genesisHash -cnotmatch '^[0-9a-f]{64}$') {
    throw 'The full-block miner requires the isolated local-release network.'
  }
  $headers['X-QC-Expected-Genesis'] = $identity.genesisHash
  Write-MinerLog 'Authenticated isolated local full-block miner started.'
  while (-not (Test-Path -LiteralPath $stopPath)) {
    $result = Invoke-MinerRequest '/api/mine/local-release' 'Post' 120 $body
    $height = 0L
    if ($result.success -isnot [bool] -or -not $result.success -or
        -not [long]::TryParse([string]$result.height, [ref]$height) -or $height -lt 1) {
      throw 'Node response did not confirm a mined block.'
    }
    Write-MinerLog "Block mined at height $height."
    if (-not (Test-Path -LiteralPath $stopPath)) { Start-Sleep -Milliseconds $DelayMs }
  }
  Write-MinerLog 'Mining client stopped.'
} catch {
  Write-MinerLog "Mining client stopped: $($_.Exception.Message)"
  exit 1
}
