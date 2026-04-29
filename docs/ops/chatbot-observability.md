# Chatbot Observability

## Metrics Endpoint
- Service exposes Prometheus metrics at `GET /metrics`.
- Local check:
  - `kubectl port-forward -n civic-connect svc/chatbot-svc 18084:8084`
  - `Invoke-RestMethod http://127.0.0.1:18084/metrics`

## Key Metrics
- `chatbot_requests_total{source,status,has_government}`
- `chatbot_request_latency_seconds{source}`
- `chatbot_fallback_total{reason}`
- `chatbot_errors_total{endpoint}`

## Suggested Alerts
- High fallback ratio:
  - `sum(rate(chatbot_fallback_total[5m])) / sum(rate(chatbot_requests_total[5m])) > 0.4`
- Elevated errors:
  - `sum(rate(chatbot_errors_total[5m])) > 0.05`
- P95 latency high:
  - `histogram_quantile(0.95, sum(rate(chatbot_request_latency_seconds_bucket[5m])) by (le)) > 5`

## Debug Drilldown
1. Query `chatbot_requests_total` split by `source`.
2. If `fallback_gemini_error` grows, inspect pod logs for Gemini errors.
3. Verify `request_government_id` and `response_source` in `/ask` response.
4. Run key rotation script if key is revoked.
