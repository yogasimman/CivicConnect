param(
  [string]$GroqModel = 'llama-3.3-70b-versatile',
  [string]$SecretsDir = "$env:APPDATA\CivicConnect\secrets"
)

$ErrorActionPreference = 'Stop'

function Get-PlainTextFromSecureString {
  param([SecureString]$Secure)
  $bstr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($Secure)
  try {
    return [System.Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
  }
  finally {
    [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
  }
}

$secretPath = Join-Path $SecretsDir 'groq_api_key.sec'
if (-not (Test-Path $secretPath)) {
  throw "Encrypted key file not found at $secretPath. Run scripts/set-groq-key.ps1 first."
}

$encrypted = (Get-Content -Path $secretPath -Raw).Trim()
$encrypted = $encrypted.Trim([char]0xFEFF)
$secureKey = ConvertTo-SecureString -String $encrypted
$plainKey = Get-PlainTextFromSecureString -Secure $secureKey

if ([string]::IsNullOrWhiteSpace($plainKey)) {
  throw 'Groq key is empty after decrypt. Run scripts/set-groq-key.ps1 again.'
}

$env:GROQ_API_KEY = $plainKey
$env:GROQ_MODEL = $GroqModel

$dockerReady = $false
for ($i = 1; $i -le 30; $i++) {
  docker info *> $null
  if ($LASTEXITCODE -eq 0) {
    $dockerReady = $true
    break
  }
  Start-Sleep -Seconds 2
}
if (-not $dockerReady) {
  throw 'Docker is not ready. Start Docker Desktop and retry.'
}

$composeDir = Split-Path $PSScriptRoot -Parent
Set-Location $composeDir

docker compose up --build -d

Write-Host 'CivicConnect stack started with GROQ env loaded from encrypted local store.'
Write-Host 'Web: http://localhost/'
Write-Host 'Chatbot health: http://localhost:8084/health'
