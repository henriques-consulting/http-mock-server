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
