param([ValidateSet('protect','unprotect')][string]$Action)
$ErrorActionPreference='Stop'
try {
 Add-Type -AssemblyName System.Security.Cryptography.ProtectedData
 $inputText=[Console]::In.ReadToEnd()
 $entropy=[Text.Encoding]::UTF8.GetBytes('QuantumCoin.Publisher.Secrets.v1')
 $scope=[Security.Cryptography.DataProtectionScope]::CurrentUser
 if($Action -eq 'protect'){
  $bytes=[Text.Encoding]::UTF8.GetBytes($inputText)
  $protected=[Security.Cryptography.ProtectedData]::Protect($bytes,$entropy,$scope)
  [Console]::Out.Write([Convert]::ToBase64String($protected))
 } else {
  $bytes=[Convert]::FromBase64String($inputText)
  $plain=[Security.Cryptography.ProtectedData]::Unprotect($bytes,$entropy,$scope)
  [Console]::Out.Write([Text.Encoding]::UTF8.GetString($plain))
 }
} catch { [Console]::Error.Write('SECRET_STORAGE_FAILED');exit 1 }
