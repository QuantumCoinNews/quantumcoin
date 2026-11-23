#requires -Version 7.3
param(
    [Parameter(Mandatory)][string]$Directory
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if (![IO.Path]::IsPathFullyQualified($Directory)) { throw 'Use an absolute external custody directory.' }
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$target = [IO.Path]::GetFullPath($Directory)
if ($target -eq $repo -or $target.StartsWith($repo + '\', [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Release key custody must be outside the source checkout.'
}
if (Test-Path -LiteralPath $target) { throw 'Choose a NEW directory; existing signing material is never replaced.' }
$parent = [IO.Path]::GetDirectoryName($target)
if (!(Test-Path -LiteralPath $parent -PathType Container)) { throw 'Prepare a private parent directory first.' }
$cursor = $parent
while ($cursor) {
    if ((Get-Item -LiteralPath $cursor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
        throw 'Indirect custody paths require separate review.'
    }
    $next = [IO.Directory]::GetParent($cursor)
    $cursor = if ($null -ne $next) { $next.FullName } else { $null }
}
$insideGit = & git -C $parent rev-parse --is-inside-work-tree 2>$null
if ($LASTEXITCODE -eq 0 -and $insideGit -eq 'true') { throw 'Signing custody must not be inside a Git worktree.' }
if ([Console]::IsInputRedirected) { throw 'Run interactively in your private Windows terminal; enter the passphrase only in the OpenSSH prompt.' }

[void][IO.Directory]::CreateDirectory($target)
$acl = [Security.AccessControl.DirectorySecurity]::new()
$acl.SetAccessRuleProtection($true, $false)
$owner = [Security.Principal.WindowsIdentity]::GetCurrent().User
foreach ($sid in @($owner, [Security.Principal.SecurityIdentifier]::new('S-1-5-18'))) {
    $rule = [Security.AccessControl.FileSystemAccessRule]::new($sid, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
    [void]$acl.AddAccessRule($rule)
}
[IO.FileSystemAclExtensions]::SetAccessControl([IO.DirectoryInfo]::new($target), $acl)
$key = Join-Path $target 'id_ed25519_quantumcoin_release'
Write-Host 'QuantumCoinNews release key: use a strong non-empty passphrase and keep its recovery copy separately.'
& ssh-keygen -t ed25519 -a 100 -C 'QuantumCoinNews-release' -f $key
if ($LASTEXITCODE -ne 0) { throw 'Release key generation failed; no trust entry was activated.' }
# An empty-passphrase probe is local and prints no key material. The real
# passphrase is entered directly in OpenSSH and never becomes a script argument.
$probe = & ssh-keygen -y -P '' -f $key 2>$null
$unprotected = $LASTEXITCODE -eq 0
$probe = $null
if ($unprotected) { throw 'Unprotected key was NOT activated. Protect it with ssh-keygen -p before adopting any public trust entry.' }
$public = (Get-Content -LiteralPath ($key + '.pub') -Raw).Trim()
if ($public -cnotmatch '^ssh-ed25519 [A-Za-z0-9+/=]+(?: .*)?$') { throw 'Unexpected release public key type.' }
$trust = Join-Path $target 'allowed-signers'
$data = [Text.Encoding]::UTF8.GetBytes('quantumcoin-release ' + $public + "`n")
$stream = [IO.File]::Open($trust, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
try { $stream.Write($data, 0, $data.Length); $stream.Flush($true) } finally { $stream.Dispose() }
& ssh-keygen -l -E sha256 -f ($key + '.pub')
if ($LASTEXITCODE -ne 0) { throw 'Public fingerprint check failed.' }
Write-Host 'Review the PUBLIC fingerprint and keep allowed-signers outside the release package. No release was signed or published.'
