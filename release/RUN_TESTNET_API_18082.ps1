param(
  [string]$DataDir,
  [string]$ReleaseDirectory,
  [string]$ReleaseCommit,
  [string]$TrustedSigners,
  [switch]$PlanOnly
)
$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'START_TESTNET_LOCAL.ps1') -Mode api @PSBoundParameters
