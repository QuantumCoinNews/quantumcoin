param(
 [string]$EnvFile,
 [string]$SecretFile,
 [string]$StateDir,
 [Parameter(Mandatory=$true)][string]$SettingsFile,
 [Parameter(Mandatory=$true)][string]$BackupDirectory,
 [switch]$Publish
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$entry=Join-Path $root 'publisher.mjs'
if(-not $StateDir){$StateDir=Join-Path $env:LOCALAPPDATA 'QuantumCoin/autopublisher'}
if(-not [IO.Path]::IsPathRooted($StateDir) -or $StateDir.Contains('"')){throw 'Valid absolute state path required'}
if([bool]$EnvFile -eq [bool]$SecretFile){throw 'Specify exactly one credential source'}
$credentialFile=if($SecretFile){$SecretFile}else{$EnvFile}
$credentialFlag=if($SecretFile){'--secret-file'}else{'--env-file'}
foreach($file in @($entry,$credentialFile,$SettingsFile)){
 if(-not [IO.Path]::IsPathRooted($file) -or $file.Contains('"') -or -not(Test-Path -LiteralPath $file -PathType Leaf)){throw 'Valid absolute entry/configuration paths required'}
}
$authorityLines=@(Get-Content -LiteralPath $SettingsFile | Where-Object {$_ -match '^QC_PUBLISHER_SCHEDULER='})
if($authorityLines.Count -ne 1){throw 'Exactly one scheduler authority is required'}
if((Get-Content -LiteralPath $SettingsFile -Raw) -notmatch '(?m)^QC_PUBLISHER_SCHEDULER=windows-task\s*$'){throw 'Public settings must select windows-task authority'}
if((Get-TimeZone).Id -ne 'Turkey Standard Time'){throw 'Windows task triggers require the Istanbul host timezone'}
$name='QuantumCoin-Publisher'
$marker='QC_CANONICAL_PUBLISHER_V1'
$existing=Get-ScheduledTask -TaskName $name -ErrorAction SilentlyContinue
if($existing -and $existing.Description -notlike "$marker*"){throw 'Task name belongs to another configuration'}
$backup=[IO.Path]::GetFullPath($BackupDirectory)
New-Item -ItemType Directory -Path $backup -Force | Out-Null
$old=@('QC-Autopublisher-21','QC-Shorts-18','QC-Shorts-21','QC_Autopublisher_18')
$matched=@()
foreach($oldName in $old){
 $task=Get-ScheduledTask -TaskName $oldName -ErrorAction SilentlyContinue
 if(-not $task){continue}
 if(-not @($task.Actions | Where-Object {($_.Execute+' '+$_.Arguments+' '+$_.WorkingDirectory) -match '(?i)quantumcoin|autopublisher|run-daily|make-shorts'}).Count){throw 'Legacy task identity mismatch'}
 $export=Join-Path $backup ($oldName+'.xml')
 if(-not(Test-Path -LiteralPath $export)){Export-ScheduledTask -TaskName $oldName | Set-Content -LiteralPath $export -Encoding utf8}
 $matched+=$oldName
}
if($existing){Export-ScheduledTask -TaskName $name | Set-Content -LiteralPath (Join-Path $backup ($name+'.previous.xml')) -Encoding utf8}
$node=(Get-Command node.exe -ErrorAction Stop).Source
$arguments='-- "'+$entry+'" run '+$credentialFlag+' "'+$credentialFile+'" --settings-file "'+$SettingsFile+'" --state-dir "'+$StateDir+'"'
if($Publish){$arguments+=' --publish --scheduler windows-task'}
$action=New-ScheduledTaskAction -Execute $node -Argument $arguments -WorkingDirectory $root
$now=Get-Date
$triggers=@(foreach($hour in @(18,21)){
 $next=$now.Date.AddHours($hour)
 if($next -le $now){$next=$next.AddDays(1)}
 New-ScheduledTaskTrigger -Daily -At $next
})
$principal=New-ScheduledTaskPrincipal -UserId ([Security.Principal.WindowsIdentity]::GetCurrent().Name) -LogonType Interactive -RunLevel Limited
$settings=New-ScheduledTaskSettingsSet -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 5) -MultipleInstances IgnoreNew -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 30)
$settings.Priority=6
Register-ScheduledTask -TaskName $name -Action $action -Trigger $triggers -Settings $settings -Principal $principal -Description "$marker; English; Istanbul 18:00/21:00; mode=$($Publish.IsPresent)" -Force | Out-Null
$disabled=@();$blocked=@()
foreach($oldName in $matched){
 try {Disable-ScheduledTask -TaskName $oldName -ErrorAction Stop | Out-Null;$disabled+=$oldName}
 catch {
  $legacy=Get-ScheduledTask -TaskName $oldName -ErrorAction Stop
  $safe=$true
  foreach($a in $legacy.Actions){
   $line=$a.Execute+' '+$a.Arguments
   $flags=@([regex]::Matches($a.Arguments,'(?:^|[\s"])-{1,2}([A-Za-z][A-Za-z0-9_-]*)') | ForEach-Object {$_.Groups[1].Value.ToLowerInvariant()})
   foreach($flag in $flags){
    if('publish'.StartsWith($flag) -or 'command'.StartsWith($flag) -or 'encodedcommand'.StartsWith($flag) -or $flag -eq 'ec'){$safe=$false}
   }
   if($line -notmatch '(?i)run-daily\.(ps1|cmd)'){$safe=$false}
   if($line -notmatch [regex]::Escape($root)){$safe=$false}
  }
  if(-not $safe){Disable-ScheduledTask -TaskName $name -ErrorAction Stop | Out-Null;throw 'Legacy task could not be disabled or proven offline; canonical task paused'}
  $blocked+=$oldName
 }
}
$info=Get-ScheduledTaskInfo -TaskName $name
[pscustomobject]@{TaskName=$name;Publish=$Publish.IsPresent;DisabledLegacy=$disabled;ProtectedLegacyOfflineOnly=$blocked;NextRunTime=$info.NextRunTime;RequiresLoggedOnUser=$true} | ConvertTo-Json -Depth 3
