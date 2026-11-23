param(
  [string]$BaseUrl = 'http://127.0.0.1:8082',
  [string[]]$Endpoint = @('/api/health'),
  [int]$Requests = 20,
  [int]$Concurrency = 2,
  [string]$Method = 'GET',
  [string]$AdminToken = $env:TELEGRAM_GAME_ADMIN_TOKEN,
  [switch]$AllowRemote,
  [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'
$version = 'QC_LOAD_TEST_SCRIPTS_SAFE_V1'
$uri = $null
if (-not [Uri]::TryCreate($BaseUrl, [UriKind]::Absolute, [ref]$uri) -or
    $uri.Scheme -notin @('http', 'https') -or $uri.UserInfo -or $uri.Query -or $uri.Fragment -or
    $uri.AbsolutePath -notmatch '^/*$') {
  throw 'BaseUrl must be an HTTP(S) origin without a path, credentials, query or fragment.'
}
if (-not $uri.IsLoopback -and (-not $AllowRemote -or $uri.Scheme -ne 'https')) {
  throw 'Remote load targets require HTTPS and explicit -AllowRemote.'
}
if ($Method -ine 'GET') {
  throw 'This load test only accepts GET.'
}

$baseTrim = $uri.GetLeftPart([UriPartial]::Authority)
$Requests = [Math]::Clamp($Requests, 1, 500)
$Concurrency = [Math]::Clamp($Concurrency, 1, 25)
$safePaths = @('/api/health', '/api/network/label', '/health', '/api/qc/health',
               '/api/admin/panel/registry', '/api/admin/monitoring/runbook')
$safeEndpoints = @(
  foreach ($entry in $Endpoint) {
    $clean = '/' + $entry.TrimStart('/')
    if ($entry.StartsWith('//') -or $clean -cnotmatch '^(/[a-z_/-]+)(?:\?((?:limit|offset)=[0-9]{1,6}(?:&(?:limit|offset)=[0-9]{1,6})?))?$' -or
        $Matches[1] -cnotin $safePaths -or ($Matches[2] -and $Matches[1] -cne '/api/admin/panel/registry')) {
      throw 'Endpoint is outside the reviewed read-only allowlist.'
    }
    $clean
  }
)
if (-not $safeEndpoints.Count) {
  throw 'At least one read-only endpoint is required.'
}

$plan = [ordered]@{
  ok = $true
  version = $version
  mode = 'SAFE_READ_ONLY_LOAD_TEST'
  baseUrl = $baseTrim
  method = 'GET'
  requests = $Requests
  concurrency = $Concurrency
  endpoints = $safeEndpoints
  allowRemote = [bool]$AllowRemote
  planOnly = [bool]$PlanOnly
  safeBoundary = [ordered]@{
    onlyGET = $true
    blocksMutatingEndpoints = $true
    doesNotMint = $true
    doesNotSendFunds = $true
    doesNotChangeBalances = $true
    doesNotChangeChain = $true
    doesNotChangeSupply = $true
    doesNotStartMining = $true
    doesNotStopMining = $true
    localOnlyByDefault = $true
  }
}
if ($PlanOnly) {
  $plan | ConvertTo-Json -Depth 8
  return
}

$targets = @(
  foreach ($endpointPath in $safeEndpoints) {
    $headers = @{}
    if ($endpointPath -ceq '/api/health') {
      if ([string]::IsNullOrWhiteSpace($env:QC_API_TOKEN)) { throw 'Node health requires QC_API_TOKEN in the environment.' }
      $headers['X-QC-API-KEY'] = $env:QC_API_TOKEN.Trim()
    } elseif ($endpointPath.StartsWith('/api/admin/')) {
      if ([string]::IsNullOrWhiteSpace($AdminToken)) { throw 'Admin reads require TELEGRAM_GAME_ADMIN_TOKEN.' }
      $headers['X-Admin-Token'] = $AdminToken.Trim()
    }
    @{ endpoint = $endpointPath; headers = $headers }
  }
)

$results = @(1..$Requests | ForEach-Object -Parallel {
  $ErrorActionPreference = 'Stop'
  $items = $using:targets
  $target = $items[($_ - 1) % $items.Count]
  $url = ($using:baseTrim) + $target.endpoint
  $watch = [Diagnostics.Stopwatch]::StartNew()
  $status = 0
  $errorLabel = ''
  try {
    $reply = Invoke-WebRequest -Uri $url -Headers $target.headers -Method Get -TimeoutSec 10 -MaximumRedirection 0 -SkipHttpErrorCheck -NoProxy
    $status = [int]$reply.StatusCode
    if ($status -lt 200 -or $status -ge 300) { $errorLabel = "HTTP $status" }
  } catch {
    $errorLabel = 'request failed'
  }
  [pscustomobject]@{
    ok = $errorLabel -eq ''
    endpoint = $target.endpoint
    status = $status
    ms = [int]$watch.ElapsedMilliseconds
    error = $errorLabel
  }
} -ThrottleLimit $Concurrency)

$latencies = @($results | Select-Object -ExpandProperty ms | Sort-Object)
$okCount = @($results | Where-Object ok).Count
$summary = [ordered]@{
  ok = $okCount -eq $Requests
  version = $version
  baseUrl = $baseTrim
  requests = $Requests
  concurrency = $Concurrency
  okCount = $okCount
  failCount = $Requests - $okCount
  averageMs = if ($latencies.Count) { [Math]::Round(($latencies | Measure-Object -Average).Average, 2) } else { 0 }
  p95Ms = if ($latencies.Count) { $latencies[[int][Math]::Ceiling($latencies.Count * 0.95) - 1] } else { 0 }
  results = $results
  safeBoundary = $plan.safeBoundary
}
$summary | ConvertTo-Json -Depth 8
if (-not $summary.ok) { exit 1 }
