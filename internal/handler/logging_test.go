package handler

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware_CapturesRequestAndResponse(t *testing.T) {
	// Capture logs
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	// Simple handler to respond
	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-From", "next")
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte(`{"ok":true}`))
		},
	)

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodPost, "/log-test?x=1", strings.NewReader("payload"))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Validate response to ensure middleware didn't interfere
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, rr.Code)
	}
	if rr.Header().Get("X-From") != "next" {
		t.Fatalf("expected response header X-From=next, got %q", rr.Header().Get("X-From"))
	}
	if !strings.Contains(rr.Body.String(), `{"ok":true}`) {
		t.Fatalf("expected response body to contain json")
	}

	// Validate log output contains key pieces
	out := buf.String()
	expectedParts := []string{
		"HTTP REQUEST",
		"Remote IP: 127.0.0.1:12345",
		"Request:",
		"    Method: POST",
		"    URI: /log-test?x=1",
		"    Headers:",
		"        Content-Type: application/json",
		"    Body: payload",
		"Response:",
		"    Status: 418",
		"    Headers:",
		"        X-From: next",
		`    Body: {"ok":true}`,
	}
	for _, part := range expectedParts {
		if !strings.Contains(out, part) {
			t.Fatalf("expected log to contain %q, got:\n%s", part, out)
		}
	}
}
