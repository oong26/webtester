# WebTester Specification

## Product Overview

WebTester is a self-hosted API and web tool for testing web application security posture and load performance.

---

## Core Capabilities

### 1. Security Scanning

| Scan Type | Engine | Description | Required Inputs |
| --- | --- | --- | --- |
| **Baseline** | Built-in Go passive checker | Header auditing, TLS, cookie security, tech leak detection. Non-intrusive. | Target URL, Ownership confirmation |
| **API** | Built-in OpenAPI parser & fuzzer | Parses OpenAPI 3.0/3.1 specs, generates fuzzing payloads for declared endpoints. | Target URL, OpenAPI Spec URL, Ownership confirmation |
| **Full** | OWASP ZAP (Docker) | Active spider crawling + active vulnerability probing (SQLi, XSS, SSRF, Path Traversal). | Target URL, Ownership confirmation |
| **BOLA** | Built-in Authorization Probe | Tests Broken Object Level Authorization across two token identities (`User A` vs `User B`). | Auth Header, User A Token, User B Token, Target Probe URLs |

#### Security Finding Severity Levels
- `High`: Immediate exploitability (SQL Injection, Remote Code Exec, BOLA)
- `Medium`: Significant risk (Reflected XSS, Missing CSRF, Insecure Direct Object)
- `Low`: Weak configuration (Missing HSTS, Insecure Cookies)
- `Info`: Information disclosure (Server banners, Verbose headers)

---

### 2. HTTP Load Testing

- **Engine:** In-process token-bucket paced worker pool (`internal/loadtest/runner.go`).
- **Target Rate:** 1 to 5,000 requests per second.
- **Worker Concurrency:** 1 to 500 concurrent goroutine workers.
- **Duration:** 1 to 600 seconds with strict timeout cancellation.
- **HTTP Methods:** `GET`, `HEAD`, `OPTIONS`, `POST`, `PUT`, `PATCH`, `DELETE`.
- **Custom Payloads:** Custom headers (RFC 7230 validated) and raw request body.
- **Real-Time Telemetry:**
  - Progress percentage
  - Total requests, successful (2xx/3xx), failed (4xx/5xx/network errors)
  - Achieved request rate (RPS)
  - Latency metrics: Min, Mean, Max, P50, P90, P95, P99
  - Status code breakdown map (`200`, `404`, `500`, etc.)
  - Network error breakdown map (`connection refused`, `i/o timeout`, etc.)

---

### 3. Authentication & Authorization

- **First-Run Onboarding:**
  - If user database table is empty, `GET /api/v1/auth/status` returns `setup_required: true`.
  - Initial setup endpoint `POST /api/v1/auth/setup` creates first administrator.
  - Subsequent calls return `409 Conflict`.
- **JWT Authentication:**
  - Standard HS256 JWT tokens.
  - Short-lived Access Token (15 minutes).
  - Long-lived Refresh Token (7 days).
- **Redis Cache & Session Storage:**
  - Refresh tokens stored in Redis under `refresh:<user_id>` with 7-day TTL.
  - Access tokens blacklisted in Redis on logout under `blacklist:<token>` with remaining TTL.
  - Dynamic rate limiting and IP/User session storage helpers.
- **Route Protection:**
  - HTTP middleware intercepts requests, extracts bearer token from `Authorization` header.
  - Verifies token signature, expiration, and blacklist status in Redis cache.
  - Injects `UserContextKey` and `TokenContextKey` into request context.
  - Public routes: `/healthz`, `/api/v1/auth/*` (status, setup, login, refresh), static web assets.
  - Protected routes: `/api/v1/scans/*`, `/api/v1/load-tests/*`, `/api/v1/auth/me`, `/api/v1/auth/logout`.
