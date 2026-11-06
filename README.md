# WebTester

A small Go tool with a web UI for testing apps you own: endpoint assertions, basic security checks and a simple load test.

> Only run this against systems you own or have written permission to test.

## Run

```bash
go run .                      # UI on http://127.0.0.1:8080
go run . -addr 127.0.0.1:9000 # custom port
```

Keep it bound to localhost. The server sends requests to whatever URL it is given, so it should not be exposed to a network.

## Build

```bash
go build -o webtester .
./webtester
```

## What it checks

| Group | Checks |
|---|---|
| Endpoint tests | method, path, headers, body, expected status, body contains, max latency |
| Security | missing security headers, version disclosure, CORS misconfig, exposed paths (`/.env`, `/.git/HEAD`, `/debug/pprof/`...), malformed input (bad JSON, oversized body, SQLi/XSS/traversal probes, error and stack-trace leaks), auth required (no token / invalid token), rate limiting |
| Load | N requests with C workers, req/s, p50/p95/p99, error rate |

## Layout

```
main.go                      server entry, embeds web/
web/index.html               single-page UI
internal/checks/             Check interface + all checks (add new ones here)
internal/runner/             builds checks from a request and aggregates a report
internal/server/             POST /api/run (same-origin JSON only)
```

## Add a check

Implement the interface in `internal/checks/checks.go`:

```go
type Check interface {
    Name() string
    Run(ctx context.Context, t Target) []Result
}
```

Then wire it up in `internal/runner/runner.go` and add a toggle in `web/index.html`.

## API

`POST /api/run` with `Content-Type: application/json`:

```json
{
  "base_url": "http://localhost:3000",
  "headers": {"Authorization": "Bearer ..."},
  "confirm_ownership": true,
  "endpoints": [{"name": "health", "method": "GET", "path": "/health", "expect_status": 200}],
  "security": {"headers": true, "cors": true, "exposed_paths": true, "malformed_input": true,
               "probe_path": "/", "auth_paths": ["/api/users"], "rate_limit_path": "/api/login"},
  "load": {"path": "/health", "total": 200, "concurrency": 10}
}
```

Returns a report with a summary and one result per assertion or finding.

## Tests

```bash
go test ./...
```
