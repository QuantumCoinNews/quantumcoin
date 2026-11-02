

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
$DesktopCmd = Join-Path -Path ([Environment]::GetFolderPath("Desktop")) -ChildPath "QuantumCoin_Telegram_Game_GUI.cmd"
Write-QcInfo ""
Write-QcInfo "===== QUANTUMCOIN TELEGRAM GAME DEV =====" -ForegroundColor Cyan
Write-QcInfo "Opening desktop GUI launcher..." -ForegroundColor Yellow
if (!(Test-Path $DesktopCmd)) { throw "Desktop launcher missing: $DesktopCmd" }
Start-Process $DesktopCmd









