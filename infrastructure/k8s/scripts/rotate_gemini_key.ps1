param(
  [Parameter(Mandatory = $true)]
  [string]$GeminiApiKey,
  [string]$Namespace = 'civic-connect',
  [string]$SecretName = 'civic-gemini-secret'
)

if ([string]::IsNullOrWhiteSpace($GeminiApiKey)) {
  throw 'Gemini API key cannot be empty.'
}

Write-Host "Updating secret $SecretName in namespace $Namespace"
kubectl create secret generic $SecretName `
  --namespace $Namespace `
  --from-literal=GEMINI_API_KEY=$GeminiApiKey `
  --dry-run=client -o yaml | kubectl apply -f -

Write-Host 'Restarting chatbot-service deployment'
kubectl rollout restart deployment/chatbot-service -n $Namespace
kubectl rollout status deployment/chatbot-service -n $Namespace --timeout=180s

Write-Host 'Smoke testing /health and /ask endpoints'
$health = Invoke-RestMethod -Method Get -Uri 'http://127.0.0.1:9090/api/v1/chatbot/health'
$askBody = @{ message = 'help'; user_id = 'rotation-smoke' } | ConvertTo-Json
$ask = Invoke-RestMethod -Method Post -Uri 'http://127.0.0.1:9090/api/v1/chatbot/ask' -Body $askBody -ContentType 'application/json'

Write-Host ("health=" + ($health | ConvertTo-Json -Compress))
Write-Host ("response_source=" + $ask.response_source + " latency_ms=" + $ask.latency_ms)
