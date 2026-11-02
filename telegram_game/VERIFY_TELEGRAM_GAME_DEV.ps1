

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
Write-QcInfo ""
Write-QcInfo "===== VERIFY TELEGRAM GAME DEV =====" -ForegroundColor Cyan
Write-QcInfo ""
Write-QcInfo "--- QC API HEALTH ---" -ForegroundColor Yellow
Invoke-RestMethod "http://127.0.0.1:8082/api/health" | ConvertTo-Json -Depth 8
Write-QcInfo ""
Write-QcInfo "--- GAME WALLET dev_user ---" -ForegroundColor Yellow
Invoke-RestMethod "http://127.0.0.1:8080/api/wallet?playerId=dev_user" | ConvertTo-Json -Depth 12
Write-QcInfo ""
Write-QcInfo "--- BRIDGE STATUS dev_user ---" -ForegroundColor Yellow
Invoke-RestMethod "http://127.0.0.1:8080/api/bridge/status?playerId=dev_user" | ConvertTo-Json -Depth 12
Write-QcInfo ""
Write-QcInfo "--- TGWT LEDGER LAST 5 ---" -ForegroundColor Yellow
Invoke-RestMethod "http://127.0.0.1:8080/api/v1/tgwt/ledger?user_id=dev_user&limit=5" | ConvertTo-Json -Depth 12








