# WebTester

Request-centric security scanner: ZAP-backed scans plus custom passive and BOLA checks.

## Run

    go mod tidy
    go build ./...
    go run ./cmd/app

Requires Docker (for ZAP) and network access on first `go mod tidy`.

## Endpoints

- `POST /api/v1/scans` — start a scan (`baseline`, `api`, `full`)
- `GET  /api/v1/scans/{scanId}` — poll status and progress
- `GET  /api/v1/scans/{scanId}/findings` — normalized findings

## Example

    curl -s -X POST localhost:8080/api/v1/scans \
      -H 'content-type: application/json' \
      -d '{
        "target_url": "http://localhost:8000",
        "openapi_url": "http://localhost:8000/api/v1/openapi.yaml",
        "scan_type": "api",
        "confirm_ownership": true,
        "auth": {
          "header_name": "Authorization",
          "user_a": "Bearer AAA",
          "user_b": "Bearer BBB"
        },
        "bola_probe_urls": ["http://localhost:8000/api/v1/farms/1"]
      }'

## Layout

- `cmd/app` — entrypoint
- `api/` — OpenAPI contract
- `internal/clients/zap` — ZAP client (Docker now, REST later)
- `internal/checks` — custom passive + BOLA rules
- `internal/services` — orchestration
- `internal/stores` — SQLite persistence
- `internal/handlers` — HTTP layer
- `pkg/` — shared errors, response, middleware
