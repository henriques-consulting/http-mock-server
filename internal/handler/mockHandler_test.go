package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"http-mock-server/internal/config"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func performRequest(
	h http.Handler,
	method, path string,
	headers map[string]string,
	body []byte,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestMockHandler_ExactMatch(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path: "/foo",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Method: "GET",
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       map[string]any{"message": "ok"},
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	rr := performRequest(
		h, http.MethodGet, "/foo", map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		}, nil,
	)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Expect JSON body
	var obj map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &obj); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if obj["message"] != "ok" {
		t.Fatalf("expected body {message: ok}, got %v", obj)
	}
}

func TestMockHandler_WildcardAny(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path: "/bar",
				Headers: map[string]string{
					"Content-Type": ".*",
				},
				Method: "POST",
				Response: config.ResponseSpec{
					StatusCode: 201,
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	rr := performRequest(
		h, http.MethodPost, "/bar", map[string]string{
			"Content-Type": "text/plain",
		}, []byte("hello"),
	)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestMockHandler_WildcardSubtype(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path: "/baz",
				Headers: map[string]string{
					"Content-Type": "application/.*",
				},
				Method: "PUT",
				Response: config.ResponseSpec{
					StatusCode: 204,
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	rr := performRequest(
		h, http.MethodPut, "/baz", map[string]string{
			"Content-Type": "application/json",
		}, nil,
	)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestMockHandler_NotFound(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path: "/q",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Method: "GET",
				Response: config.ResponseSpec{
					StatusCode: 200,
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	rr := performRequest(
		h, http.MethodGet, "/not-exists", map[string]string{
			"Content-Type": "text/plain",
		}, nil,
	)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestMockHandler_ResponseHeaders(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path: "/headers",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Method: "GET",
				Response: config.ResponseSpec{
					StatusCode: 200,
					Headers: map[string]string{
						"X-Test": "yes",
					},
					Body: "ok",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	rr := performRequest(
		h, http.MethodGet, "/headers", map[string]string{
			"Content-Type": "text/plain",
		}, nil,
	)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Header().Get("X-Test") != "yes" {
		t.Fatalf("expected header X-Test=yes, got %q", rr.Header().Get("X-Test"))
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("expected body ok, got %q", rr.Body.String())
	}
}

func TestMockHandler_QueryParamExactMatch(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/search",
				Method: "GET",
				QueryParams: map[string]string{
					"foo": "bar",
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "found",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	rr := performRequest(h, http.MethodGet, "/search?foo=bar", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "found" {
		t.Fatalf("expected body 'found', got %q", rr.Body.String())
	}
}

func TestMockHandler_QueryParamRegexMatch(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/users",
				Method: "GET",
				QueryParams: map[string]string{
					"id": "[0-9]+",
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "user found",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	// Should match numeric id
	rr := performRequest(h, http.MethodGet, "/users?id=123", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should not match non-numeric id
	rr = performRequest(h, http.MethodGet, "/users?id=abc", nil, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestMockHandler_QueryParamMultiple(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/search",
				Method: "GET",
				QueryParams: map[string]string{
					"q":    ".*",
					"page": "[0-9]+",
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "results",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	// Should match when all params match
	rr := performRequest(h, http.MethodGet, "/search?q=test&page=1", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should not match when one param doesn't match
	rr = performRequest(h, http.MethodGet, "/search?q=test&page=abc", nil, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestMockHandler_QueryParamMissing(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/api",
				Method: "GET",
				QueryParams: map[string]string{
					"token": "secret",
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "authorized",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	// Should not match when required param is missing
	rr := performRequest(h, http.MethodGet, "/api", nil, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	// Should match when param is present and correct
	rr = performRequest(h, http.MethodGet, "/api?token=secret", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestMockHandler_QueryParamNoRuleMatchesAny(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/open",
				Method: "GET",
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "open endpoint",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	// Should match without query params
	rr := performRequest(h, http.MethodGet, "/open", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should also match with any query params
	rr = performRequest(h, http.MethodGet, "/open?foo=bar&baz=qux", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestMockHandler_QueryParamExtraParamsAllowed(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/test",
				Method: "GET",
				QueryParams: map[string]string{
					"required": "value",
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	// Should match even with extra params not in rule
	rr := performRequest(h, http.MethodGet, "/test?required=value&extra=ignored&another=also", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestMockHandler_QueryParamInvalidRegexFallsBackToExact(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/test",
				Method: "GET",
				QueryParams: map[string]string{
					"pattern": "[invalid(regex",
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	// Should fall back to exact match for invalid regex
	rr := performRequest(h, http.MethodGet, "/test?pattern=[invalid(regex", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected exact match fallback, got status %d", rr.Code)
	}
}

func TestMockHandler_ResponseDelayFixed(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/delayed",
				Method: "GET",
				ResponseDelay: &config.ResponseDelay{
					Min: 100,
					Max: 100,
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "delayed response",
				},
			},
		},
	}

	// Use deterministic random for testing
	r := rand.New(rand.NewSource(42))
	h := NewMockHandlerWithRand(cfg, r)

	start := time.Now()
	rr := performRequest(h, http.MethodGet, "/delayed", nil, nil)
	elapsed := time.Since(start)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should have delayed at least 100ms
	if elapsed < 100*time.Millisecond {
		t.Fatalf("expected delay of at least 100ms, got %v", elapsed)
	}
}

func TestMockHandler_ResponseDelayRange(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/delayed-range",
				Method: "GET",
				ResponseDelay: &config.ResponseDelay{
					Min: 50,
					Max: 150,
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "delayed response",
				},
			},
		},
	}

	// Use deterministic random for testing
	r := rand.New(rand.NewSource(42))
	h := NewMockHandlerWithRand(cfg, r)

	start := time.Now()
	rr := performRequest(h, http.MethodGet, "/delayed-range", nil, nil)
	elapsed := time.Since(start)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should have delayed at least 50ms (the minimum)
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected delay of at least 50ms, got %v", elapsed)
	}

	// Should not exceed max + generous buffer for CI systems
	if elapsed > 250*time.Millisecond {
		t.Fatalf("delay exceeded expected range, got %v", elapsed)
	}
}

func TestMockHandler_NoDelayWhenNotConfigured(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/fast",
				Method: "GET",
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "fast response",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	start := time.Now()
	rr := performRequest(h, http.MethodGet, "/fast", nil, nil)
	elapsed := time.Since(start)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should complete quickly without delay (allowing generous buffer for CI systems)
	if elapsed > 100*time.Millisecond {
		t.Fatalf("expected fast response, but took %v", elapsed)
	}
}

func TestMockHandler_CalculateDelay(t *testing.T) {
	cfg := &config.Config{}
	r := rand.New(rand.NewSource(42))
	h := NewMockHandlerWithRand(cfg, r)

	tests := []struct {
		name     string
		delay    *config.ResponseDelay
		minMs    int
		maxMs    int
	}{
		{
			name:  "fixed delay",
			delay: &config.ResponseDelay{Min: 100, Max: 100},
			minMs: 100,
			maxMs: 100,
		},
		{
			name:  "range delay",
			delay: &config.ResponseDelay{Min: 50, Max: 150},
			minMs: 50,
			maxMs: 150,
		},
		{
			name:  "zero delay",
			delay: &config.ResponseDelay{Min: 0, Max: 0},
			minMs: 0,
			maxMs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := h.calculateDelay(tt.delay)
			ms := int(duration.Milliseconds())
			if ms < tt.minMs || ms > tt.maxMs {
				t.Errorf("calculateDelay() = %dms, want between %d and %d", ms, tt.minMs, tt.maxMs)
			}
		})
	}
}

func TestMockHandler_ConcurrentDelayedRequests(t *testing.T) {
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/concurrent",
				Method: "GET",
				ResponseDelay: &config.ResponseDelay{
					Min: 10,
					Max: 20,
				},
				Response: config.ResponseSpec{
					StatusCode: 200,
					Body:       "ok",
				},
			},
		},
	}

	h := NewMockHandler(cfg)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr := performRequest(h, http.MethodGet, "/concurrent", nil, nil)
			if rr.Code != http.StatusOK {
				errors <- fmt.Errorf("unexpected status: %d", rr.Code)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
