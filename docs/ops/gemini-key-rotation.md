# Gemini Key Rotation Playbook

## Prerequisites
- New valid Gemini API key.
- `kubectl` access to `civic-connect` namespace.
- API gateway reachable at `http://127.0.0.1:9090`.

## Rotate Key
- PowerShell:
  - `./infrastructure/k8s/scripts/rotate_gemini_key.ps1 -GeminiApiKey '<NEW_KEY>'`

## Manual Alternative
1. Update secret:
   - `kubectl create secret generic civic-gemini-secret --namespace civic-connect --from-literal=GEMINI_API_KEY=<NEW_KEY> --dry-run=client -o yaml | kubectl apply -f -`
2. Restart chatbot deployment:
   - `kubectl rollout restart deployment/chatbot-service -n civic-connect`
3. Wait for rollout:
   - `kubectl rollout status deployment/chatbot-service -n civic-connect --timeout=180s`

## Smoke Validation
1. `GET /api/v1/chatbot/health`
2. `POST /api/v1/chatbot/ask` with body: `{"message":"help","user_id":"smoke"}`
3. Confirm response has:
   - `response`
   - `response_source`
   - `latency_ms`
