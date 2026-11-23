$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$lockDir=Join-Path $root '.build'
New-Item -ItemType Directory -Path $lockDir -Force | Out-Null
$lease=$null
try {
 try{$lease=[IO.File]::Open((Join-Path $lockDir 'runtime.lock'),[IO.FileMode]::OpenOrCreate,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)}
 catch [IO.IOException]{[Console]::Error.WriteLine('BUILD_BUSY: wait for running publisher/build');exit 73}
 $manifest=Join-Path $root 'dist/build-manifest.json'
 if(Test-Path -LiteralPath $manifest){Remove-Item -LiteralPath $manifest}
 Push-Location $root
 try{& node -- (Join-Path $root 'node_modules/typescript/bin/tsc') -p tsconfig.json;$code=$LASTEXITCODE}finally{Pop-Location}
 if($code -ne 0){exit $code}
 $sources=@(Get-ChildItem -LiteralPath (Join-Path $root 'src') -Recurse -File -Filter '*.ts')
 foreach($relative in @('package.json','package-lock.json','tsconfig.json','publisher.mjs','scripts/build.ps1','scripts/hold-build-lock.ps1')){$sources+=Get-Item -LiteralPath (Join-Path $root $relative)}
 function Get-Fingerprints($files){@($files | Sort-Object FullName | ForEach-Object {@{path=[IO.Path]::GetRelativePath($root,$_.FullName).Replace('\','/');sha256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()}})}
 $compiled=@(Get-ChildItem -LiteralPath (Join-Path $root 'dist') -Recurse -File -Filter '*.js')
 $sharedSchema=Join-Path (Split-Path -Parent $root) 'config/runtime-environment.schema.json'
 $value=@{version='QC_BUILD_V1';source=(Get-Fingerprints $sources);compiled=(Get-Fingerprints $compiled);sharedEnvironmentSchemaHash=(Get-FileHash -LiteralPath $sharedSchema -Algorithm SHA256).Hash.ToLowerInvariant()}
 $temporary=$manifest+'.'+[guid]::NewGuid().ToString('N')+'.tmp'
 $value | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $temporary -Encoding utf8
 Move-Item -LiteralPath $temporary -Destination $manifest
} finally {if($null -ne $lease){$lease.Dispose()}}
