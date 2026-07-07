package handler

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware_OmitsLargeRequestBody(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	largeBody := strings.Repeat("x", logBodyLimit+1)
	req := httptest.NewRequest(http.MethodPost, "/large-body", strings.NewReader(largeBody))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if strings.Contains(out, largeBody) {
		t.Fatal("expected large request body to be omitted from log")
	}
	if !strings.Contains(out, bodyOmittedNotice) {
		t.Fatalf("expected omission notice in log, got:\n%s", out)
	}
}

func TestLoggingMiddleware_OmitsLargeResponseBody(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	largeBody := strings.Repeat("y", logBodyLimit+1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(largeBody))
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/large-response", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Response body is still written to the client
	if rr.Body.String() != largeBody {
		t.Fatal("expected response body to be written to client unchanged")
	}

	out := buf.String()
	if strings.Contains(out, largeBody) {
		t.Fatal("expected large response body to be omitted from log")
	}
	if !strings.Contains(out, bodyOmittedNotice) {
		t.Fatalf("expected omission notice in log, got:\n%s", out)
	}
}

func TestLoggingMiddleware_BoundaryRequestBody(t *testing.T) {
	// Exactly logBodyLimit bytes should be logged in full, not omitted.
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	boundaryBody := strings.Repeat("b", logBodyLimit)
	req := httptest.NewRequest(http.MethodPost, "/boundary", strings.NewReader(boundaryBody))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if strings.Contains(out, "(omitted") {
		t.Fatal("expected body at exactly logBodyLimit bytes to be logged, not omitted")
	}
	if !strings.Contains(out, boundaryBody) {
		t.Fatalf("expected log to contain the body at the boundary, got:\n%s", out)
	}
}

func TestLoggingMiddleware_BoundaryResponseBody(t *testing.T) {
	// Exactly logBodyLimit bytes should be logged in full, not omitted.
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	boundaryBody := strings.Repeat("b", logBodyLimit)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(boundaryBody))
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/boundary", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if strings.Contains(out, "(omitted") {
		t.Fatal("expected body at exactly logBodyLimit bytes to be logged, not omitted")
	}
	if !strings.Contains(out, boundaryBody) {
		t.Fatalf("expected log to contain the body at the boundary, got:\n%s", out)
	}
}

func TestLoggingMiddleware_LargeRequestBodyPassthrough(t *testing.T) {
	// A large request body must still be fully readable by the next handler even when omitted from logs.
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	largeBody := strings.Repeat("z", logBodyLimit+1)
	var receivedBody []byte
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodPost, "/passthrough", strings.NewReader(largeBody))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if string(receivedBody) != largeBody {
		t.Fatalf("next handler received %d bytes, expected %d", len(receivedBody), len(largeBody))
	}
}

func TestLoggingMiddleware_EmptyBody(t *testing.T) {
	// Nil/empty request and response bodies should log "(empty)".
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if strings.Count(out, "(empty)") < 2 {
		t.Fatalf("expected both request and response bodies to log as (empty), got:\n%s", out)
	}
}

func TestLoggingMiddleware_PrettyPrintsJSONRequestBody(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodPost, "/json", strings.NewReader(`{"name":"a","nested":{"x":1}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	pretty := "    Body:\n        {\n          \"name\": \"a\",\n          \"nested\": {\n            \"x\": 1\n          }\n        }"
	if !strings.Contains(out, pretty) {
		t.Fatalf("expected pretty-printed JSON request body, got:\n%s", out)
	}
}

func TestLoggingMiddleware_PrettyPrintsJSONResponseBody(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	pretty := "    Body:\n        {\n          \"ok\": true\n        }"
	if !strings.Contains(out, pretty) {
		t.Fatalf("expected pretty-printed JSON response body, got:\n%s", out)
	}
}

func TestLoggingMiddleware_PrettyPrintsJSONWithCharsetParameter(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodPost, "/json", strings.NewReader(`{"ok":true}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if !strings.Contains(out, "    Body:\n        {\n          \"ok\": true\n        }") {
		t.Fatalf("expected charset-parameterized JSON to be pretty-printed, got:\n%s", out)
	}
}

func TestLoggingMiddleware_MalformedJSONFallsBackToRaw(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	malformed := `{"ok":true`
	req := httptest.NewRequest(http.MethodPost, "/json", strings.NewReader(malformed))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if !strings.Contains(out, "    Body: "+malformed) {
		t.Fatalf("expected malformed JSON to log raw on a single line, got:\n%s", out)
	}
}

func TestLoggingMiddleware_MalformedJSONResponseFallsBackToRaw(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	malformed := `{"ok":true`
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(malformed))
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/bad-json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if !strings.Contains(out, "    Body: "+malformed) {
		t.Fatalf("expected malformed JSON response body to log raw, got:\n%s", out)
	}
}

func TestLoggingMiddleware_OmitsLargeJSONBody(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	largeBody := strings.Repeat("x", logBodyLimit+1)
	req := httptest.NewRequest(http.MethodPost, "/large-json", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if !strings.Contains(out, bodyOmittedNotice) {
		t.Fatalf("expected omission notice for large JSON body, got:\n%s", out)
	}
	if strings.Contains(out, "    Body:\n        {") {
		t.Fatal("expected no pretty-print block when body exceeds size limit")
	}
}

func TestLoggingMiddleware_BoundaryJSONBody(t *testing.T) {
	// A JSON-content-typed body at exactly logBodyLimit must be logged, not
	// omitted. Guards the > vs >= threshold now that logBodyLimit is the only
	// bound on JSON pretty-printing. The body is intentionally invalid JSON so
	// the raw fallback is taken and the assertion stays on the omission boundary.
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	body := strings.Repeat("x", logBodyLimit)
	req := httptest.NewRequest(http.MethodPost, "/boundary-json", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if strings.Contains(out, "(omitted") {
		t.Fatal("expected JSON body at exactly logBodyLimit bytes to be logged, not omitted")
	}
}

func TestLoggingMiddleware_PrettyPrintsLargeJSONWithinLimit(t *testing.T) {
	// A large JSON body (well over 64 KiB but within logBodyLimit) is still
	// pretty-printed; logBodyLimit is the only bound.
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	filler := strings.Repeat("a", 256*1024)
	body := `{"data":"` + filler + `"}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodPost, "/big-json", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if !strings.Contains(out, "    Body:\n        {") {
		t.Fatalf("expected large within-limit JSON to be pretty-printed as an indented block, got body section start:\n%s", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, `"data": "`) {
		t.Fatal("expected pretty-printed JSON to add a space after the key colon")
	}
	if strings.Contains(out, "    Body: {") {
		t.Fatal("expected the pretty-print path, not the inline raw path")
	}
}

func TestLoggingMiddleware_ResponseBodyPartialTruncation(t *testing.T) {
	// Two writes where the second straddles the capture limit: the client gets
	// both chunks, and the log shows the omission notice.
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	firstChunk := strings.Repeat("a", logBodyLimit)
	secondChunk := "overflow"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(firstChunk))
		_, _ = w.Write([]byte(secondChunk))
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/two-writes", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Body.Len() != logBodyLimit+len(secondChunk) {
		t.Fatalf("expected client to receive %d bytes, got %d", logBodyLimit+len(secondChunk), rr.Body.Len())
	}
	out := buf.String()
	if !strings.Contains(out, bodyOmittedNotice) {
		t.Fatalf("expected omission notice when body exceeds limit across two writes, got:\n%s", out)
	}
}

func TestLoggingMiddleware_NonJSONContentTypeNotPrettyPrinted(t *testing.T) {
	var buf bytes.Buffer
	oldOut := log.Writer()
	defer log.SetOutput(oldOut)
	log.SetOutput(&buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	jsonLooking := `{"ok":true}`
	req := httptest.NewRequest(http.MethodPost, "/text", strings.NewReader(jsonLooking))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	out := buf.String()
	if !strings.Contains(out, "    Body: "+jsonLooking) {
		t.Fatalf("expected non-JSON content type to log body inline, got:\n%s", out)
	}
}

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
