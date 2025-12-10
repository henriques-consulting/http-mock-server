package handler

import (
	"encoding/json"
	"fmt"
	"http-mock-server/internal/config"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// MockHandler handles mock requests based on configuration
type MockHandler struct {
	config *config.Config
}

// NewMockHandler creates a new mock handler
func NewMockHandler(cfg *config.Config) *MockHandler {
	return &MockHandler{config: cfg}
}

func (h *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rule := h.findMatchingRule(r)
	if rule == nil {
		http.NotFound(w, r)
		return
	}

	h.writeResponse(w, rule)
}

func (h *MockHandler) findMatchingRule(r *http.Request) *config.RequestRule {
	path := r.URL.Path
	method := strings.ToUpper(r.Method)

	for i := range h.config.Requests {
		rule := &h.config.Requests[i]

		if rule.Path != path {
			continue
		}

		if rule.Method != method {
			continue
		}

		if !h.matchesHeaders(rule.Headers, r.Header) {
			continue
		}

		if !h.matchesQueryParams(rule.QueryParams, r.URL.Query()) {
			continue
		}

		if !h.matchesBody(rule.Body, r) {
			continue
		}

		return rule
	}

	return nil
}

func (h *MockHandler) writeResponse(w http.ResponseWriter, rule *config.RequestRule) {
	// Set response headers
	for key, value := range rule.Response.Headers {
		w.Header().Set(key, value)
	}

	// Set status code
	w.WriteHeader(rule.Response.StatusCode)

	// Write body if present
	if rule.Response.Body == nil {
		return
	}

	if err := h.writeBody(w, rule.Response.Body); err != nil {
		// Log error but don't change response since headers are already written
		fmt.Printf("Error writing response body: %v\n", err)
	}
}

func (h *MockHandler) writeBody(w http.ResponseWriter, body interface{}) error {
	switch v := body.(type) {
	case string:
		_, err := w.Write([]byte(v))
		return err
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = w.Write(data)
		return err
	}
}

func (h *MockHandler) matchesHeaders(ruleHeaders map[string]string, requestHeaders http.Header) bool {
	// If no headers are specified in the rule, it matches any request
	if len(ruleHeaders) == 0 {
		return true
	}

	// All rule headers must match
	for headerName, headerPattern := range ruleHeaders {
		requestValue := requestHeaders.Get(headerName)

		// Compile the regex pattern
		pattern, err := regexp.Compile(headerPattern)
		if err != nil {
			// If pattern is invalid, treat as exact match
			if requestValue != headerPattern {
				return false
			}
		} else {
			// Use regex matching
			if !pattern.MatchString(requestValue) {
				return false
			}
		}
	}

	return true
}

func (h *MockHandler) matchesQueryParams(ruleParams map[string]string, requestParams url.Values) bool {
	// If no query params are specified in the rule, it matches any request
	if len(ruleParams) == 0 {
		return true
	}

	// All rule query params must match
	for paramName, paramPattern := range ruleParams {
		requestValue := requestParams.Get(paramName)

		// Compile the regex pattern
		pattern, err := regexp.Compile(paramPattern)
		if err != nil {
			// If pattern is invalid, treat as exact match
			if requestValue != paramPattern {
				return false
			}
		} else {
			// Use regex matching
			if !pattern.MatchString(requestValue) {
				return false
			}
		}
	}

	return true
}

func (h *MockHandler) matchesBody(ruleBody string, r *http.Request) bool {
	if ruleBody == "" {
		return true
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	pattern, err := regexp.Compile(ruleBody)
	if err != nil {
		return false
	}

	return pattern.Match(body)
}
