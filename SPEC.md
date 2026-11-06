# WebTester

A self-hosted testing toolkit for backend APIs, written in Go, with a built-in web UI. It runs three kinds of tests from one place:

1. **Functional endpoint tests**: does each endpoint return what it should?
2. **Baseline security checks**: does the app avoid the common, easy-to-fix mistakes?
3. **Basic load tests**: how fast does it respond under concurrent traffic?

> **Authorized use only.** Run this tool only against systems you own or have written permission to test. Testing someone else's system without authorization is illegal in most places.

---

## Purpose

WebTester is for developers who want a quick way to check their own backend during development, before a release, or in a staging environment, without setting up several separate tools.

It answers questions like:

- Does `GET /api/users` return `200` and the expected body within 300 ms?
- Do my protected routes reject requests with no token or a bad token?
- Am I sending sensible security headers?
- Does my API leak stack traces or SQL errors when given garbage input?
- Does anything sensitive (`/.env`, `/.git/HEAD`, `/debug/pprof/`) respond with `200`?
- What are my p50/p95/p99 latencies at 10 concurrent users?

### What it is not

- It is **not a full penetration-testing tool**. Its security checks are heuristics for common misconfigurations. It does not replace OWASP ZAP, Burp Suite, or a professional security assessment.
- It is **not a full load-testing platform**. For large or long-running load tests use tools like k6, vegeta or wrk.
- It is **not a replacement for unit or integration tests** inside your codebase. It tests the running app from the outside, over HTTP.
- A passing result does **not** prove your app is secure. It only means these specific checks found nothing.

---

## Quick start

Requires **Go 1.22+**. There are no third-party dependencies.

```bash
go run .                        # UI at http://127.0.0.1:8080
go run . -addr 127.0.0.1:9000   # custom address

# or build a binary
go build -o webtester .
./webtester
```

Then in the UI:

1. Enter the **Base URL** of the app under test (for example `http://localhost:8000`).
2. Optionally add **default headers** as JSON (for example `{"Authorization": "Bearer ..."}`).
3. Add endpoint tests and/or enable security checks and the load test.
4. Tick the authorization checkbox and click **Run tests**.
5. Review the results and optionally **Export JSON**.

Tip: security checks run against the **Probe path**. Use a real route of your app (for example an `/api/...` route if you added middleware only to an API group), not just `/`.

---

## Specification

### Endpoint tests

Each test sends one request and asserts on the response.

| Field | Description | Default |
|---|---|---|
| `name` | Label shown in the report | |
| `method` | HTTP method | `GET` |
| `path` | Request path | |
| `headers` | Extra headers for this request | |
| `body` | Request body (`Content-Type: application/json` is added if you don't set one) | |
| `expect_status` | Expected HTTP status | `200` |
| `expect_body_contains` | Substring the response body must contain | |
| `max_latency_ms` | Maximum allowed response time | no limit |

A failed assertion is reported as **medium**. A request that cannot be made at all (connection refused, timeout) is **high**.

### Security checks

| Check | What it does | Findings and severity |
|---|---|---|
| **Security headers** | Requests the probe path and inspects response headers | `Content-Security-Policy` missing: medium. CSP sent only as `Report-Only`: low. `Strict-Transport-Security` missing (HTTPS targets only): medium. `X-Content-Type-Options` (must be `nosniff`), `X-Frame-Options` (or CSP `frame-ancestors`), `Referrer-Policy` missing: low. `X-Powered-By` present, or a `Server` header containing a version number: low |
| **CORS** | Sends `Origin: https://evil.example` | Origin reflected with credentials allowed: high. Origin reflected: medium. Wildcard `*`: low |
| **Exposed paths** | Requests common sensitive paths and flags `200` responses. Responses identical to a random 404 baseline are ignored, so single-page apps don't cause false positives | `/.env`, `/.git/HEAD`, `/config.json`, `/backup.zip`, `/debug/pprof/`: high. `/actuator`, `/metrics`, `/admin`, `/server-status`: medium. `/swagger.json`, `/openapi.json`, `/.DS_Store`: low |
| **Malformed input** | Sends 6 probes to the probe path: truncated JSON, wrong content type, 256 KB body, SQLi-style query, XSS payload, path traversal | Response leaks internals (`panic:`, `goroutine`, SQL error text, `.go:` file references) or returns file contents: high. XSS payload reflected unescaped, or a `5xx` response: medium |
| **Auth required** | Calls each listed protected path with no credentials and with an invalid bearer token (`GET`) | A `2xx` response: high |
| **Rate limit** | Sends a burst of 60 requests (20 at a time) to one path | No `429` seen: low |

Notes:

- Default headers are applied to every check except **Auth required**, which deliberately sends none (or an invalid token).
- Redirects are **not followed**, so checks see the raw response.

### Load test

| Setting | Description | Default | Limit |
|---|---|---|---|
| `path` | Path to hit (`GET`) | | |
| `total` | Total requests | 200 | 5000 |
| `concurrency` | Parallel workers | 10 | 100 |

Reports requests per second, p50/p95/p99 latency, status-code counts and error rate. Errors are transport failures and `5xx` responses. The test passes if the error rate is below 1%, and fails as **medium** otherwise.

### Limits and timeouts

| Item | Value |
|---|---|
| Per-request timeout | 15 s |
| Whole run | 3 minutes |
| Response body read per request | 1 MB |
| Endpoint tests per run | 200 |
| API request body | 1 MB |

---

## Result format

Every check returns one or more results:

```json
{
  "check": "header: Content-Security-Policy",
  "passed": false,
  "severity": "low",
  "message": "report-only: violations are reported but the policy is not enforced",
  "detail": "Content-Security-Policy-Report-Only: default-src 'self' (switch to ...)",
  "duration_ms": 12,
  "metrics": {}
}
```

- `severity` is one of `info`, `low`, `medium`, `high`. Passing results are `info`.
- `metrics` is only used by the load test (`rps`, `p50_ms`, `p95_ms`, `p99_ms`, `error_rate_pct`).

A full report wraps the results with an `id`, `target`, `started_at`, `duration_ms` and a `summary` (`total`, `passed`, `failed`, `high`, `medium`, `low`).

---

## HTTP API

The UI is a thin client over one endpoint, so you can also script it.

`POST /api/run` with `Content-Type: application/json`:

```json
{
  "base_url": "http://localhost:8000",
  "headers": {"Authorization": "Bearer <token>"},
  "confirm_ownership": true,
  "endpoints": [
    {"name": "health", "method": "GET", "path": "/health", "expect_status": 200}
  ],
  "security": {
    "headers": true,
    "cors": true,
    "exposed_paths": true,
    "malformed_input": true,
    "probe_path": "/api/health",
    "auth_paths": ["/api/users"],
    "rate_limit_path": "/api/login"
  },
  "load": {"path": "/api/health", "total": 200, "concurrency": 10}
}
```

Returns the report described above. Errors return `{"error": "..."}` with status `400`, `403` or `415`. The run is synchronous and has no stored history.

---

## Safety design

- **Localhost by default.** The server binds to `127.0.0.1:8080`. Keep it that way: it sends requests to whatever URL it is given, so it should not be exposed to a network.
- **Same-origin JSON only.** `POST /api/run` rejects non-JSON content types and requests whose `Origin` differs from the host, so other websites cannot make your browser drive the tool.
- **Explicit confirmation.** A run is refused unless `confirm_ownership` is `true` (the checkbox in the UI).
- **Bounded traffic.** Load and burst sizes are capped, and a run stops after 3 minutes.
- **Local storage note.** The UI saves your form (including default headers such as tokens) in your browser's `localStorage` for convenience. Use a short-lived test token, and clear site data if you share the machine. The authorization checkbox is never saved.

---

## Project layout

```
main.go                     server entry point, embeds web/
web/index.html              single-page UI
internal/checks/
  checks.go                 Check interface, Target, Result, HTTP helper
  functional.go             endpoint assertions
  security.go               headers, CORS, exposed paths, malformed input, auth, rate limit
  load.go                   load test and percentiles
  checks_test.go            unit tests (httptest servers)
internal/runner/runner.go   builds checks from a request, runs them, aggregates the report
internal/server/server.go   POST /api/run and static file serving
```

## Adding a check

Implement the interface in `internal/checks/checks.go`:

```go
type Check interface {
    Name() string
    Run(ctx context.Context, t Target) []Result
}
```

Then:

1. Add the check in a new or existing file under `internal/checks/`.
2. Add a config field in `internal/runner/runner.go` and append the check in `Run`.
3. Add a toggle or input for it in `web/index.html`.
4. Add a test using `httptest.NewServer`.

## Tests

```bash
go vet ./...
go test ./...
```

## Known limitations

- Security checks are heuristics: they can produce false positives and miss real vulnerabilities.
- Auth checks use `GET` only and treat any non-`2xx` response as "rejected".
- Authentication flows (log in first, reuse the token) are not supported; supply a token via default headers.
- Results are not persisted between runs, and runs cannot be cancelled from the UI.
- The load test is intentionally simple (closed-loop, `GET` only).

## Ideas for later

Not implemented yet: saved test suites as YAML files, a login-then-test flow, HTML report export, comparing two runs, and CI mode with a non-zero exit code on failures.