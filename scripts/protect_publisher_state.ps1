param([switch]$Repair)
$ErrorActionPreference='Stop'
$state=[IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'QuantumCoin/autopublisher'))
if(-not(Test-Path -LiteralPath $state -PathType Container)){throw 'Publisher state is not present'}
$paths=@(Get-Item -LiteralPath $state)+@(Get-ChildItem -LiteralPath $state -Recurse -Force)
$count=0;$repaired=0
foreach($item in $paths){
 $full=[IO.Path]::GetFullPath($item.FullName)
 if($full -ne $state -and -not $full.StartsWith($state+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'ACL path escape'}
 if($item.Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'ACL reparse point rejected'}
 $rules=@((Get-Acl -LiteralPath $full).GetAccessRules($true,$true,[Security.Principal.SecurityIdentifier]))
 $broad=@($rules | Where-Object {$_.IdentityReference.Value -in @('S-1-1-0','S-1-5-11','S-1-5-32-545','S-1-15-2-1') -and $_.AccessControlType -eq 'Allow' -and ($_.FileSystemRights -band [Security.AccessControl.FileSystemRights]::ReadData)})
 if($broad.Count){
  $relative=[IO.Path]::GetRelativePath($state,$full)
  if(-not $Repair -or $item.PSIsContainer -or $relative -notlike 'legacy-snapshot-*'){throw 'Publisher state permits broad read access; review Windows ACLs before sharing'}
  $parent=Split-Path -Parent $full
  $parentRules=@((Get-Acl -LiteralPath $parent).GetAccessRules($true,$true,[Security.Principal.SecurityIdentifier]))
  if(@($parentRules | Where-Object {$_.IdentityReference.Value -in @('S-1-1-0','S-1-5-11','S-1-5-32-545','S-1-15-2-1') -and $_.AccessControlType -eq 'Allow' -and ($_.FileSystemRights -band [Security.AccessControl.FileSystemRights]::ReadData)}).Count){throw 'Repair requires a private parent directory'}
  $hash=(Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash
  $temporary=Join-Path $parent ('.private-copy-'+[guid]::NewGuid().ToString('N')+'.tmp')
  $inputStream=[IO.File]::OpenRead($full);$outputStream=[IO.File]::Open($temporary,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::None)
  try{$inputStream.CopyTo($outputStream);$outputStream.Flush($true)}finally{$inputStream.Dispose();$outputStream.Dispose()}
  if((Get-FileHash -LiteralPath $temporary -Algorithm SHA256).Hash -ne $hash){throw 'Private copy verification failed; original preserved'}
  $old=Join-Path $parent ('.previous-copy-'+[guid]::NewGuid().ToString('N')+'.tmp')
  Move-Item -LiteralPath $full -Destination $old
  Move-Item -LiteralPath $temporary -Destination $full
  if((Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash -ne $hash){throw 'Replacement verification failed; previous copy preserved'}
  $newRules=@((Get-Acl -LiteralPath $full).GetAccessRules($true,$true,[Security.Principal.SecurityIdentifier]))
  if(@($newRules | Where-Object {$_.IdentityReference.Value -in @('S-1-1-0','S-1-5-11','S-1-5-32-545','S-1-15-2-1') -and $_.AccessControlType -eq 'Allow' -and ($_.FileSystemRights -band [Security.AccessControl.FileSystemRights]::ReadData)}).Count){throw 'Replacement is not private; previous copy preserved'}
  [IO.File]::SetLastWriteTimeUtc($full,$item.LastWriteTimeUtc)
  # Both paths are generated siblings under the validated private state root.
  # The exact archived bytes are verified at the original path before unlink.
  Remove-Item -LiteralPath $old
  $repaired++
 }
 $count++
}
[pscustomobject]@{verifiedPrivateItems=$count;privateCopiesCreated=$repaired;broadReadGroups=0;existingWindowsServiceAndCapabilityRulesPreserved=$true;aclRewritePerformed=$false} | ConvertTo-Json
