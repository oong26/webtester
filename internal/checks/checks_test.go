package checks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	mux.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Write([]byte("secret data"))
	})
	mux.HandleFunc("/open", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("no auth needed"))
	})
	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "panic: runtime error: index out of range", http.StatusInternalServerError)
	})
	return httptest.NewServer(mux)
}

func target(url string) Target {
	return Target{BaseURL: url, Headers: map[string]string{"Authorization": "Bearer good"}, Client: NewClient(5 * time.Second)}
}

func TestEndpointTest(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	ctx := context.Background()

	r := EndpointTest{Title: "ok", Path: "/ok", ExpectBodyContains: "hello"}.Run(ctx, target(srv.URL))
	if len(r) != 1 || !r[0].Passed {
		t.Fatalf("expected pass, got %+v", r)
	}
	r = EndpointTest{Title: "wrong status", Path: "/ok", ExpectStatus: 201}.Run(ctx, target(srv.URL))
	if r[0].Passed {
		t.Fatalf("expected failure for wrong status, got %+v", r)
	}
}

func TestAuthRequired(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res := AuthRequired{Paths: []string{"/secret", "/open"}}.Run(context.Background(), target(srv.URL))
	passed := map[string]bool{}
	for _, r := range res {
		passed[r.Check] = r.Passed
	}
	if !passed["auth: /secret (no credentials)"] || !passed["auth: /secret (invalid token)"] {
		t.Fatalf("/secret should be rejected: %+v", res)
	}
	if passed["auth: /open (no credentials)"] {
		t.Fatalf("/open should be flagged: %+v", res)
	}
}

func TestSecurityHeadersFlagsMissing(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res := SecurityHeaders{Path: "/ok"}.Run(context.Background(), target(srv.URL))
	failed := 0
	for _, r := range res {
		if !r.Passed {
			failed++
		}
	}
	if failed < 3 {
		t.Fatalf("expected several missing headers, got %d: %+v", failed, res)
	}
}

func TestMalformedInputDetectsLeak(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res := MalformedInput{Path: "/crash"}.Run(context.Background(), target(srv.URL))
	for _, r := range res {
		if r.Passed {
			t.Fatalf("every probe against /crash should fail: %+v", r)
		}
	}
}

func TestLoadTest(t *testing.T) {
	srv := testServer()
	defer srv.Close()
	res := LoadTest{Path: "/ok", Total: 50, Concurrency: 5}.Run(context.Background(), target(srv.URL))
	if len(res) != 1 || !res[0].Passed || res[0].Metrics["rps"] <= 0 {
		t.Fatalf("unexpected load result: %+v", res)
	}
}

func cspResult(t *testing.T, headers map[string]string) Result {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Write([]byte("x"))
	}))
	defer srv.Close()
	for _, r := range (SecurityHeaders{Path: "/"}).Run(context.Background(), target(srv.URL)) {
		if r.Check == "header: Content-Security-Policy" {
			return r
		}
	}
	t.Fatal("no CSP result returned")
	return Result{}
}

func TestCSPEnforced(t *testing.T) {
	r := cspResult(t, map[string]string{"Content-Security-Policy": "default-src 'none'"})
	if !r.Passed {
		t.Fatalf("enforced CSP should pass: %+v", r)
	}
}

func TestCSPReportOnly(t *testing.T) {
	r := cspResult(t, map[string]string{"Content-Security-Policy-Report-Only": "default-src 'self'"})
	if r.Passed || r.Severity != Low || !strings.Contains(r.Message, "report-only") {
		t.Fatalf("report-only CSP should be a low 'report-only' finding: %+v", r)
	}
}

func TestCSPMissing(t *testing.T) {
	r := cspResult(t, nil)
	if r.Passed || r.Severity != Medium || r.Message != "missing" {
		t.Fatalf("absent CSP should be a medium 'missing' finding: %+v", r)
	}
}
