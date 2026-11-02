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
cd C:\Projects\quantumcoin\telegram_game
$env:QC_API_BASE = "http://127.0.0.1:18082"
$env:QUANTUMCOIN_API_BASE = "http://127.0.0.1:18082"
$env:QC_NODE_API = "http://127.0.0.1:18082"
$env:QC_GAME_MINING_MODE = "TESTNET_SIMULATED"
$env:TELEGRAM_GAME_MINING_MODE = "TESTNET_SIMULATED"
Write-QcInfo ""
Write-QcInfo "===== TELEGRAM GAME SERVER 8080 - TESTNET SIMULATED NO-RELOAD =====" -ForegroundColor Cyan
Write-QcInfo "QC_API_BASE=$env:QC_API_BASE" -ForegroundColor Yellow
Write-QcInfo "QC_GAME_MINING_MODE=$env:QC_GAME_MINING_MODE" -ForegroundColor Yellow
Write-QcInfo "TELEGRAM_GAME_MINING_MODE=$env:TELEGRAM_GAME_MINING_MODE" -ForegroundColor Yellow
python -c "import os; print('ENV QC_GAME_MINING_MODE =', os.getenv('QC_GAME_MINING_MODE')); from telegram_game.dev_api import _get_game_mining_mode; print('PY MODE =', _get_game_mining_mode())"
python -m uvicorn telegram_game.dev_api:app --host 127.0.0.1 --port 8080











