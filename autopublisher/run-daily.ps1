param(
 [ValidateSet('1','2')][string]$Slot,
 [switch]$Publish,
 [switch]$RenderOnly,
 [string]$Date,
 [string]$StateDir,
 [string]$SecretFile,[string]$EnvFile,[string]$SettingsFile,
 [string]$Platforms
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$entry=Join-Path $PSScriptRoot 'publisher.mjs'
if(-not(Test-Path -LiteralPath $entry)){[Console]::Error.WriteLine('BUILD_REQUIRED: run npm ci and npm run build in autopublisher.');exit 1}
$forward=@('--',$entry,'run')
if($Slot){$forward+=@('--slot',$Slot)}
if($Publish){$forward+='--publish'}
if($RenderOnly){$forward+='--render-only'}
if($Date){$forward+=@('--date',$Date)}
if($StateDir){$forward+=@('--state-dir',$StateDir)}
if($SecretFile){$forward+=@('--secret-file',$SecretFile)}
if($EnvFile){$forward+=@('--env-file',$EnvFile)}
if($SettingsFile){$forward+=@('--settings-file',$SettingsFile)}
if($Platforms){$forward+=@('--platforms',$Platforms)}
& node @forward
exit $LASTEXITCODE
