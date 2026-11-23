param([Parameter(Mandatory=$true)][string]$StateDirectory)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$repo=[IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$publisher=Join-Path $repo 'autopublisher'
if((Get-Item -LiteralPath $publisher -Force).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Publisher root reparse point rejected'}
$state=[IO.Path]::GetFullPath($StateDirectory)
if(-not [IO.Path]::IsPathRooted($StateDirectory) -or $state.Equals($repo,[StringComparison]::OrdinalIgnoreCase) -or $state.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'State must be outside the checkout'}
$ledger=Join-Path $state 'publish/ledger.json'
if(Test-Path -LiteralPath $ledger){$v=Get-Content -LiteralPath $ledger -Raw | ConvertFrom-Json;if($v.PSObject.Properties['activeJob'] -and $v.activeJob){throw 'Finish or review the active job before migration'}}
$archive=Join-Path $state ('legacy-snapshot-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $archive -Force | Out-Null
$files=@()
foreach($relative in @('assets/pool','logs','tmp','out','src/content/out','src/content/queue','src/content/archive')){
 $folder=Join-Path $publisher $relative
 if(Test-Path -LiteralPath $folder){if((Get-Item -LiteralPath $folder -Force).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Migration directory reparse point rejected'};$files+=@(Get-ChildItem -LiteralPath $folder -Recurse -File -Force)}
}
foreach($relative in @('assets/bg.mp4','last.json','autopublisher.lock','src/content/pool/pool_state.json')){
 $file=Join-Path $publisher $relative;if(Test-Path -LiteralPath $file){$files+=Get-Item -LiteralPath $file}
}
$files+=@(Get-ChildItem -LiteralPath $publisher -Force -File | Where-Object {$_.Name -like '*.log' -or $_.Name -like '.tmp-*'})
$records=@();$media=0
foreach($file in ($files | Sort-Object FullName -Unique)){
 $source=[IO.Path]::GetFullPath($file.FullName)
 if(-not $source.StartsWith($publisher+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'Unsafe migration source'}
 $relative=[IO.Path]::GetRelativePath($publisher,$source)
 $destination=[IO.Path]::GetFullPath((Join-Path $archive $relative))
 if(-not $destination.StartsWith($archive+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Migration path escape'}
 $hash=(Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash
 if($relative -match '^assets[\\/]pool[\\/](audio|bgs)[\\/](.+)$'){
  $canonical=Join-Path $state ('media/'+$Matches[1]+'/'+$Matches[2])
  New-Item -ItemType Directory -Path (Split-Path -Parent $canonical) -Force | Out-Null
  if(Test-Path -LiteralPath $canonical){if((Get-FileHash -LiteralPath $canonical -Algorithm SHA256).Hash -ne $hash){throw 'Existing canonical media differs'}}else{Copy-Item -LiteralPath $source -Destination $canonical}
  if((Get-FileHash -LiteralPath $canonical -Algorithm SHA256).Hash -ne $hash){throw 'Media copy verification failed'}
  $media++
 }
 New-Item -ItemType Directory -Path (Split-Path -Parent $destination) -Force | Out-Null
 Move-Item -LiteralPath $source -Destination $destination -ErrorAction Stop
 if((Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash -ne $hash){throw 'Archive verification failed'}
 $records+=@{source=$source;archive=$destination;sha256=$hash}
 $records | ConvertTo-Json -Depth 3 | Set-Content -LiteralPath (Join-Path $archive 'migration-manifest.json') -Encoding utf8
}
[pscustomobject]@{archivedFiles=$records.Count;canonicalMediaFiles=$media;sourceFilesDeletedWithoutArchive=0;credentialsTouched=$false} | ConvertTo-Json
