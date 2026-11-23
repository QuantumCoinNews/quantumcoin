param(
    [switch]$PlanOnly,
    [string]$ProjectDirectory = (Split-Path -Parent $PSScriptRoot),
    [string]$OutputDirectory = (Join-Path $env:USERPROFILE 'Downloads'),
    [string]$ArchiveSet = ('QuantumCoin-Source-' + (Get-Date -Format yyyyMMdd))
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$project = (Resolve-Path -LiteralPath $ProjectDirectory).Path
$output = (Resolve-Path -LiteralPath $OutputDirectory).Path
if ($ArchiveSet -cnotmatch '^[A-Za-z0-9._-]+$') { throw 'Invalid archive set name' }
$setName = $ArchiveSet
$utf8 = [Text.UTF8Encoding]::new($false)
Write-Output 'Collecting source inventory'

function ExclusionReason([string]$relative) {
    $p = $relative.Replace('\', '/').ToLowerInvariant()
    $name = [IO.Path]::GetFileName($p)
    if ($p -match '(^|/)(\.git|node_modules|__pycache__|\.venv|venv|env|vendor|\.pytest_cache|\.mypy_cache|\.ruff_cache|\.build|\.gocache|\.gotmp|secrets|\.wallet|logs|out|tmp|coverage)(/|$)') { return 'dependencies-runtime-or-private-directory' }
    if ($p -match '(^|/)(_legacy_runtime_[^/]*|release_clean_test[^/]*|release_backup_[^/]*|_local_audit_reports|_local_audit_backups|_wm_review|__qcdiag|old_working_release|_checkpoints|_root_exe_quarantine|offline-recovery|_backups)(/|$)') { return 'historical-local-directory' }
    if ($p -match '^(bin|build)/|^cmd/wallet/build/|^qc-desktop/|^release/dist/|^i18n/bin/|^webwallet/dist/|^autopublisher/(dist|output|outputs|media|videos|images|cache)/') { return 'generated-or-old-release' }
    if ($p -like 'assets/user closing conversation*') { return 'private-conversation' }
    if ($name -ne '.env.example' -and $name -notlike '*.env.example' -and ($name -match '(^|\.)env($|\.)|^env\.local$')) { return 'environment-secret' }
    if ($p -eq 'config.json' -or $name -match '^\.(wallet|chain\.json|bonus\.json)$') { return 'local-runtime-config' }
    if ($p -eq 'telegram_game/requirements.lock') { return '' }
    if ($name -match '\.(db|sqlite|sqlite3|dat|log)([.-]|$)|\.(exe|dll|pdb|so|dylib|o|obj|syso|pyc|pyo|pyd|zip|7z|rar|tar|gz|pem|key|pfx|p12|qcwallet|test|out|backup|tmp|lock)$') { return 'database-binary-secret-or-archive' }
    if ($name -match '\.before-|\.bak($|[_.-])|\.old$|\.recovery-plan\.local\.json$|\.custody\.local\.json$') { return 'local-backup' }
    $sourceName = [IO.Path]::GetExtension($name) -in @('.go','.py','.js','.mjs','.cjs','.ts','.tsx','.jsx','.ps1','.sh','.cmd','.bat','.md','.html','.css','.svg')
    if (!$sourceName -and $name -match '^(chain_data|security_events|wallet_data|wallet_balance|bonus_store|ai_state|ai_security_reviews|mined_balance|admin_settlement_report)([_.-]|$)') { return 'runtime-state' }
    if (!$sourceName -and $name -match '^(miner_(address|pid|cmd_pid|heartbeat)|wallet_address|wallet_priv|miner_priv|wallet_recovery_words|wallet_recovery_manifest|_last_gui_send_receiver)([_.-]|$)') { return 'wallet-or-operator-state' }
    if ([IO.Path]::GetExtension($name) -notin @('.go', '.py', '.js', '.mjs', '.cjs', '.ts', '.tsx', '.jsx', '.ps1', '.sh', '.cmd', '.bat', '.md', '.html', '.css', '.svg', '.disabled')) {
        if ($name -match 'wallet.*priv|miner.*priv|private[-_]?key|wallet_recovery|client[-_]?secret|oauth.*token|credentials|secrets|recovery.*words|seed.*phrase|^token\.json$|^private\.json$|\.dpapi\.') { return 'credential-or-wallet-secret' }
    }
    if ($p -match '^autopublisher/src/content/(out|queue|archive)/|^autopublisher/src/content/pool/pool_state\.json$|(^|/)\.(monitoring|advisory)(/|$)|\.advisory/|\.security/') { return 'service-state' }
    if ($name -match '^(_flat|_tree|_goenv|_gomod|project_structure|_?build_errors|lint_report|cmd-tree|project-tree|release-tree|walletminer-tree|go_env|tree|block_around_141|symbols)\.txt$') { return 'machine-diagnostic' }
    return ''
}

$start = [Diagnostics.ProcessStartInfo]::new()
$start.FileName = (Get-Command python -CommandType Application | Select-Object -First 1).Source
foreach ($argument in @((Join-Path $project 'scripts/source_hygiene.py'), '--repo', $project, '--inventory')) { $start.ArgumentList.Add($argument) }
$start.RedirectStandardOutput = $true
$start.RedirectStandardError = $true
$start.StandardOutputEncoding = [Text.Encoding]::UTF8
$start.UseShellExecute = $false
$start.CreateNoWindow = $true
$process = [Diagnostics.Process]::Start($start)
$listed = $process.StandardOutput.ReadToEnd()
$gitError = $process.StandardError.ReadToEnd()
$process.WaitForExit()
if ($process.ExitCode -ne 0) { throw 'Canonical source inventory failed' }
Write-Output 'Checking source paths and required files'
$candidates = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
foreach ($path in ($listed | ConvertFrom-Json).sourcePaths) { [void]$candidates.Add($path.Replace('\', '/')) }

$files = [Collections.Generic.List[object]]::new()
$omissions = @{}
$credentialPattern = '(?:sk-proj-[A-Za-z0-9_-]{40,}|github_pat_[A-Za-z0-9_]{40,}|ghp_[A-Za-z0-9]{30,}|AKIA[A-Z0-9]{16}|-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----)'
$textExtensions = @('.go','.py','.js','.mjs','.cjs','.ts','.tsx','.jsx','.json','.md','.txt','.html','.css','.ps1','.sh','.cmd','.bat','.yml','.yaml','.toml','.ini','.example','.disabled')
foreach ($relative in $candidates) {
    $reason = ExclusionReason $relative
    $absolute = [IO.Path]::GetFullPath((Join-Path $project $relative))
    if (!$absolute.StartsWith($project + '\', [StringComparison]::OrdinalIgnoreCase)) { throw 'Inventory path escapes project' }
    if ($reason) { throw "Canonical source would be omitted ($reason): $relative" }
    if (!(Test-Path -LiteralPath $absolute -PathType Leaf)) { continue }
    $file = Get-Item -LiteralPath $absolute -Force
    $cursor = $file.FullName
    while ($cursor -and $cursor -ne $project) {
        if (([IO.File]::GetAttributes($cursor) -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Indirect source path requires review: $relative" }
        $cursor = [IO.Path]::GetDirectoryName($cursor)
    }
    if ([IO.Path]::GetExtension($relative).ToLowerInvariant() -in $textExtensions -and $file.Length -lt 16MB) {
        $content = [IO.File]::ReadAllText($absolute)
        if ([regex]::IsMatch($content, $credentialPattern)) { throw "Potential credential material requires review in $relative" }
    }
    $files.Add([pscustomobject]@{ Path=$relative; FullName=$absolute; Size=$file.Length; Written=$file.LastWriteTimeUtc.Ticks })
}
foreach ($required in @('go.mod', 'go.sum', 'main.go', 'LICENSE', 'blockchain/mined_balance.go', 'wallet/miner_address.go', 'monitoring/engine.go', 'configs/testnet.env.example', 'telegram_game/requirements.lock', 'telegram_game/tests/test_wallet_recovery_ui_v1.js', 'telegram_game/telegram_game/dev_ui/game.html', 'telegram_game/telegram_game/dev_ui/wallet_recovery.js', 'autopublisher/package-lock.json', 'webwallet/package-lock.json')) {
    if ($required -notin $files.Path) { throw "Required source omitted: $required" }
}
if ($files.Count -lt 100) { throw 'Unexpectedly small source inventory' }
Write-Output ("Source inventory checked: {0} files" -f $files.Count)
$buckets = @(for ($i=0; $i -lt 4; $i++) { [pscustomobject]@{Part=$i+1; Bytes=[long]0; Files=[Collections.Generic.List[object]]::new()} })
foreach ($file in ($files | Sort-Object @{Expression='Size'; Descending=$true}, Path)) {
    $bucket = $buckets | Sort-Object Bytes,Part | Select-Object -First 1
    $bucket.Files.Add($file)
    $bucket.Bytes += $file.Size
}
$plan = [ordered]@{ Files=$files.Count; UncompressedBytes=($buckets|Measure-Object Bytes -Sum).Sum; Omissions=$omissions; Parts=@($buckets|ForEach-Object { @{Part=$_.Part; Files=$_.Files.Count; Bytes=$_.Bytes} }) }
if ($PlanOnly) { $plan | ConvertTo-Json -Depth 5; exit 0 }

$destinations = @(for($i=1; $i -le 4; $i++){ Join-Path $output ('{0}-Part-{1:D2}-of-04.zip' -f $setName,$i) })
foreach($destination in $destinations){ if(Test-Path -LiteralPath $destination){throw 'A requested ZIP destination already exists'} }
$scratch = Join-Path ([IO.Path]::GetTempPath()) ('qc-source-zips-' + [guid]::NewGuid().ToString('N'))
$null = New-Item -ItemType Directory -Path $scratch
$allRecords = [Collections.Generic.List[object]]::new()
$receipts = [Collections.Generic.List[object]]::new()
foreach ($bucket in $buckets) {
    $name = [IO.Path]::GetFileName($destinations[$bucket.Part-1])
    $temporary = Join-Path $scratch $name
    $records = [Collections.Generic.List[object]]::new()
    $fileStream = [IO.File]::Open($temporary,[IO.FileMode]::CreateNew,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
    $zip = [IO.Compression.ZipArchive]::new($fileStream,[IO.Compression.ZipArchiveMode]::Create,$false)
    try {
        foreach ($file in ($bucket.Files | Sort-Object Path)) {
            $entry = $zip.CreateEntry('quantumcoin/' + $file.Path,[IO.Compression.CompressionLevel]::Optimal)
            $entryStream = $entry.Open()
            $hasher = [Security.Cryptography.SHA256]::Create()
            $crypto = [Security.Cryptography.CryptoStream]::new($entryStream,$hasher,[Security.Cryptography.CryptoStreamMode]::Write,$true)
            $source = [IO.File]::Open($file.FullName,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::Read)
            try {
                $source.CopyTo($crypto)
                $crypto.FlushFinalBlock()
                $digest = [Convert]::ToHexString($hasher.Hash).ToLowerInvariant()
            } finally { $source.Dispose(); $crypto.Dispose(); $hasher.Dispose(); $entryStream.Dispose() }
            $current = Get-Item -LiteralPath $file.FullName -Force
            if($current.Length -ne $file.Size -or $current.LastWriteTimeUtc.Ticks -ne $file.Written){throw 'Source changed during snapshot. Retry from a stable tree.'}
            $record = [pscustomobject]@{Path=$file.Path; Size=$file.Size; SHA256=$digest}
            $records.Add($record)
            $allRecords.Add($record)
        }
        $manifest = [ordered]@{Schema='QC_REVIEW_SOURCE_ZIP_V1'; Set=$setName; Part=$bucket.Part; TotalParts=4; Layout='Extract all four parts into the same directory'; Source='Current working tree including uncommitted source'; SourcePolicy='Canonical source_hygiene inventory excludes custody, runtime and generated data before packaging; missing canonical members fail closed'; ExcludedCategories=$omissions; Files=@($records)}
        $manifestEntry = $zip.CreateEntry(('quantumcoin/SOURCE-MANIFEST-Part-{0:D2}.json' -f $bucket.Part))
        $writer = [IO.StreamWriter]::new($manifestEntry.Open(),$utf8)
        try { $writer.Write(($manifest | ConvertTo-Json -Depth 7)) } finally { $writer.Dispose() }
    } finally { $zip.Dispose(); $fileStream.Dispose() }
    $read = [IO.Compression.ZipFile]::OpenRead($temporary)
    try {
        foreach($record in $records){
            $entry = $read.GetEntry('quantumcoin/' + $record.Path)
            if(!$entry -or $entry.Length -ne $record.Size){throw 'ZIP membership or length verification failed'}
            $stream=$entry.Open(); $hash=[Security.Cryptography.SHA256]::Create()
            try { $actual=[Convert]::ToHexString($hash.ComputeHash($stream)).ToLowerInvariant() } finally { $stream.Dispose(); $hash.Dispose() }
            if($actual -ne $record.SHA256){throw 'ZIP content hash verification failed'}
        }
        if($read.Entries.Count -ne $records.Count+1){throw 'Unexpected ZIP entries'}
    } finally { $read.Dispose() }
    $receipts.Add([pscustomobject]@{Name=$name; Files=$records.Count; Bytes=(Get-Item -LiteralPath $temporary).Length; SHA256=(Get-FileHash -LiteralPath $temporary -Algorithm SHA256).Hash.ToLowerInvariant()})
    Write-Output ("Verified part {0}/4, {1} files" -f $bucket.Part,$records.Count)
}
foreach($record in $allRecords){
    $currentHash=(Get-FileHash -LiteralPath (Join-Path $project $record.Path) -Algorithm SHA256).Hash.ToLowerInvariant()
    if($currentHash -ne $record.SHA256){throw 'Source changed after packaging. No ZIP files have been published.'}
}
foreach($receipt in $receipts){
    $temporary=Join-Path $scratch $receipt.Name
    $destination=Join-Path $output $receipt.Name
    if(!(Test-Path -LiteralPath $temporary -PathType Leaf) -or (Test-Path -LiteralPath $destination)){throw 'ZIP installation precondition failed'}
    Move-Item -LiteralPath $temporary -Destination $destination
}
$receipts | ConvertTo-Json -Depth 4
