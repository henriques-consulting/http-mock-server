package config

import (
	"strings"
	"testing"
)

func TestValidateResponseDelay(t *testing.T) {
	tests := []struct {
		name        string
		delay       *ResponseDelay
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil delay is valid",
			delay:       nil,
			expectError: false,
		},
		{
			name:        "valid fixed delay",
			delay:       &ResponseDelay{Min: 100, Max: 100},
			expectError: false,
		},
		{
			name:        "valid range delay",
			delay:       &ResponseDelay{Min: 100, Max: 500},
			expectError: false,
		},
		{
			name:        "zero delay is valid",
			delay:       &ResponseDelay{Min: 0, Max: 0},
			expectError: false,
		},
		{
			name:        "negative min",
			delay:       &ResponseDelay{Min: -100, Max: 100},
			expectError: true,
			errorMsg:    "responseDelay min cannot be negative",
		},
		{
			name:        "negative max",
			delay:       &ResponseDelay{Min: 0, Max: -100},
			expectError: true,
			errorMsg:    "responseDelay max cannot be negative",
		},
		{
			name:        "min exceeds max",
			delay:       &ResponseDelay{Min: 500, Max: 100},
			expectError: true,
			errorMsg:    "responseDelay min (500) cannot exceed max (100)",
		},
		{
			name:        "max exceeds limit",
			delay:       &ResponseDelay{Min: 0, Max: 15000},
			expectError: true,
			errorMsg:    "responseDelay max (15000) exceeds maximum allowed",
		},
		{
			name:        "max at limit is valid",
			delay:       &ResponseDelay{Min: 0, Max: MaxResponseDelayMs},
			expectError: false,
		},
		{
			name:        "max at limit+1 is invalid",
			delay:       &ResponseDelay{Min: 0, Max: MaxResponseDelayMs + 1},
			expectError: true,
			errorMsg:    "exceeds maximum allowed",
		},
		{
			name:        "fixed delay at max limit is valid",
			delay:       &ResponseDelay{Min: MaxResponseDelayMs, Max: MaxResponseDelayMs},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Port: 8080},
				Requests: []RequestRule{
					{
						Path:          "/test",
						Method:        "GET",
						ResponseDelay: tt.delay,
						Response: ResponseSpec{
							StatusCode: 200,
						},
					},
				},
			}

			err := cfg.validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}
