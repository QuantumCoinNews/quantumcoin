param([ValidateSet('1','2')][string]$Slot,[switch]$Publish,[switch]$RenderOnly,[string]$Date,[string]$StateDir,[string]$SecretFile,[string]$EnvFile,[string]$SettingsFile,[string]$Platforms)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot 'run-daily.ps1') @PSBoundParameters
exit $LASTEXITCODE
