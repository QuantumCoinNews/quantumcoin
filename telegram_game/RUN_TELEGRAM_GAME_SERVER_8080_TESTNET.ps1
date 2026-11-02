

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
$null = "Stop"

Set-Location -Path "C:\Projects\quantumcoin\telegram_game"

$env:TELEGRAM_GAME_CHAIN_MODE = "testnet"
$env:TELEGRAM_GAME_NETWORK = "testnet"
$env:QC_TESTNET = "1"
$env:QC_NETWORK = "testnet"
$env:QC_CHAIN_MODE = "testnet"
$env:QC_NET_MAGIC = "QC_TESTNET_V1"

$env:QC_API_BASE = "http://127.0.0.1:18082"
$env:QUANTUMCOIN_API_BASE = "http://127.0.0.1:18082"
$env:QC_NODE_API = "http://127.0.0.1:18082"

# Telegram Game testnet mode must not call real chain mining/settlement.
$env:TELEGRAM_GAME_MINING_MODE = "TESTNET_SIMULATED"

Write-QcInfo "QC_TG_TESTNET_CONNECTION_V1" -ForegroundColor Cyan
Write-QcInfo "Telegram Game mode=$env:TELEGRAM_GAME_CHAIN_MODE" -ForegroundColor Yellow
Write-QcInfo "QC_API_BASE=$env:QC_API_BASE" -ForegroundColor Yellow
Write-QcInfo "TELEGRAM_GAME_MINING_MODE=$env:TELEGRAM_GAME_MINING_MODE" -ForegroundColor Yellow

python -m uvicorn telegram_game.dev_api:app --host 127.0.0.1 --port 8080









