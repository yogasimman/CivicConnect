param(
  [string]$GroqApiKey,
  [string]$SecretsDir = "$env:APPDATA\CivicConnect\secrets"
)

$ErrorActionPreference = 'Stop'

if (-not $GroqApiKey) {
  $secureInput = Read-Host 'Enter GROQ API key' -AsSecureString
} else {
  $secureInput = ConvertTo-SecureString -String $GroqApiKey -AsPlainText -Force
}

New-Item -ItemType Directory -Path $SecretsDir -Force | Out-Null
$secretPath = Join-Path $SecretsDir 'groq_api_key.sec'

$secureInput | ConvertFrom-SecureString | Set-Content -Path $secretPath -Encoding Ascii

Write-Host "Saved encrypted Groq key at: $secretPath"
Write-Host 'This file is encrypted with your Windows user profile (DPAPI).'
