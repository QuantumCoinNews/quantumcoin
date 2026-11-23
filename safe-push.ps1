param(
  [string]$Message = 'chore: safe update',
  [string]$Remote = 'origin',
  [string]$Branch,
  [switch]$Commit,
  [switch]$Push,
  [switch]$NoPush
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if ($Push -and $NoPush) { throw 'Choose either Push or NoPush.' }
$arguments = @((Join-Path $PSScriptRoot 'scripts/safe_push.py'), '--message', $Message, '--remote', $Remote)
if ($Branch) { $arguments += @('--branch', $Branch) }
if ($Commit) { $arguments += '--commit' }
if ($Push) { $arguments += '--push' }
& python @arguments
exit $LASTEXITCODE
