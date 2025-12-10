package handler

import (
	"bytes"
	"encoding/json"
	"http-mock-server/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
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
