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
$null = "Stop"

cd C:\Projects\quantumcoin\telegram_game

$env:QC_API_BASE = "http://127.0.0.1:8082"
$env:QUANTUMCOIN_API_BASE = "http://127.0.0.1:8082"
$env:QC_NODE_API = "http://127.0.0.1:8082"

Write-QcInfo ""
Write-QcInfo "===== TELEGRAM GAME SERVER 8080 =====" -ForegroundColor Cyan
Write-QcInfo "QC_API_BASE=$env:QC_API_BASE" -ForegroundColor Yellow
Write-QcInfo "Game URL: http://127.0.0.1:8080" -ForegroundColor Yellow
Write-QcInfo ""

python -m uvicorn telegram_game.dev_api:app --host 127.0.0.1 --port 8080 --reload










