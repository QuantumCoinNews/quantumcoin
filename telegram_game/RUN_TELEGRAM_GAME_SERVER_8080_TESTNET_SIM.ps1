# Persistence is selected by telegram_game.persistence.
# To retain a legacy DB, explicitly set QC_GAME_DB to its absolute path first.
# QC_33C7_PSSCRIPTANALYZER_LEGACY_SUPPRESSIONS
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSAvoidUsingCmdletAliases', '', Justification='QC 33C7: legacy/test/run script naming or control-flow retained to avoid behavior change while zeroing analyzer warnings.')]
param()
function Write-QcInfo {
    param(
        [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
        [object[]] $Object,
        [switch] $NoNewline,
        [object] $Separator = " ",
        [ConsoleColor] $ForegroundColor,
        [ConsoleColor] $BackgroundColor
    )

    $null = $ForegroundColor
    $null = $BackgroundColor


    $null = $NoNewline

    $text = ($Object | ForEach-Object { [string] $_ }) -join [string] $Separator

    Write-Output $text
}
# QC_TG_TESTNET_CONNECTION_V1
$env:TELEGRAM_GAME_CHAIN_MODE = "testnet"
$env:TELEGRAM_GAME_NETWORK = "testnet"
$env:QC_TESTNET = "1"
$env:QC_NETWORK = "testnet"
$env:QC_CHAIN_MODE = "testnet"
$env:QC_NET_MAGIC = "QC_TESTNET_V1"
$env:TELEGRAM_GAME_MINING_MODE = "TESTNET_SIMULATED"
$null = "Stop"
Set-Location -LiteralPath $PSScriptRoot

$env:QC_API_BASE = "http://127.0.0.1:18082"
$env:QUANTUMCOIN_API_BASE = "http://127.0.0.1:18082"
$env:QC_NODE_API = "http://127.0.0.1:18082"
$env:QC_GAME_MINING_MODE = "TESTNET_SIMULATED"
Write-QcInfo ""
Write-QcInfo "===== TELEGRAM GAME SERVER 8080 - TESTNET SIMULATED =====" -ForegroundColor Cyan
Write-QcInfo "QC_API_BASE=$env:QC_API_BASE" -ForegroundColor Yellow
Write-QcInfo "QC_GAME_MINING_MODE=$env:QC_GAME_MINING_MODE" -ForegroundColor Yellow
Write-QcInfo "Game URL: http://127.0.0.1:8080" -ForegroundColor Yellow
Write-QcInfo ""
python (Join-Path $PSScriptRoot "../scripts/build_wallet_crypto.py")
if ($LASTEXITCODE -ne 0) { throw "QC local wallet asset build failed; server not started" }
python -m uvicorn telegram_game.runtime:app --host 127.0.0.1 --port 8080 --no-proxy-headers --reload
