package config

import (
	"strings"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr string
	}{
		// No unit → bytes
		{"0", 0, ""},
		{"1", 1, ""},
		{"512", 512, ""},
		{"1000b", 1000, ""},
		{"1000B", 1000, ""},
		// Kilobytes
		{"1k", 1024, ""},
		{"1K", 1024, ""},
		{"1kb", 1024, ""},
		{"1KB", 1024, ""},
		{"10 kb", 10240, ""},
		// Megabytes
		{"1m", 1024 * 1024, ""},
		{"1mb", 1024 * 1024, ""},
		{"1MB", 1024 * 1024, ""},
		{"2 MB", 2 * 1024 * 1024, ""},
		// Gigabytes
		{"1g", 1024 * 1024 * 1024, ""},
		{"1gb", 1024 * 1024 * 1024, ""},
		{"1GB", 1024 * 1024 * 1024, ""},
		// Whitespace tolerance
		{"  10  ", 10, ""},
		{"2  mb", 2 * 1024 * 1024, ""},
		// Errors
		{"", 0, "cannot be empty"},
		{"abc", 0, "must start with a number"},
		{"-1", 0, "cannot be negative"},
		{"10xyz", 0, "unknown unit"},
		{"10 bytes", 0, "unknown unit"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseSize(%q) expected error containing %q, got nil", tt.input, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseSize(%q) error = %q, want containing %q", tt.input, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSize(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parseSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateRandomBody(t *testing.T) {
	tests := []struct {
		name        string
		randomBody  *RandomBodySpec
		body        interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil randomBody is valid",
			randomBody:  nil,
			expectError: false,
		},
		{
			name:        "valid plaintext",
			randomBody:  &RandomBodySpec{Type: "plaintext", Size: "512"},
			expectError: false,
		},
		{
			name:        "valid json",
			randomBody:  &RandomBodySpec{Type: "json", Size: "2 KB"},
			expectError: false,
		},
		{
			name:        "valid xml",
			randomBody:  &RandomBodySpec{Type: "xml", Size: "1 KB"},
			expectError: false,
		},
		{
			name:        "zero-size plaintext is valid",
			randomBody:  &RandomBodySpec{Type: "plaintext", Size: "0"},
			expectError: false,
		},
		{
			name:        "size at max limit is valid",
			randomBody:  &RandomBodySpec{Type: "plaintext", Size: "2 GB"},
			expectError: false,
		},
		{
			name:        "size above max is invalid",
			randomBody:  &RandomBodySpec{Type: "plaintext", Size: "3 GB"},
			expectError: true,
			errorMsg:    "exceeds maximum allowed",
		},
		{
			name:        "invalid type",
			randomBody:  &RandomBodySpec{Type: "html", Size: "100"},
			expectError: true,
			errorMsg:    "randomBody type must be one of",
		},
		{
			name:        "empty type",
			randomBody:  &RandomBodySpec{Type: "", Size: "100"},
			expectError: true,
			errorMsg:    "randomBody type must be one of",
		},
		{
			name:        "negative size",
			randomBody:  &RandomBodySpec{Type: "plaintext", Size: "-1"},
			expectError: true,
			errorMsg:    "cannot be negative",
		},
		{
			name:        "unknown unit",
			randomBody:  &RandomBodySpec{Type: "plaintext", Size: "10 bytes"},
			expectError: true,
			errorMsg:    "unknown unit",
		},
		{
			name:        "body and randomBody both set",
			randomBody:  &RandomBodySpec{Type: "plaintext", Size: "100"},
			body:        "some body",
			expectError: true,
			errorMsg:    "body and randomBody are mutually exclusive",
		},
		{
			name:        "json size too small",
			randomBody:  &RandomBodySpec{Type: "json", Size: "1"},
			expectError: true,
			errorMsg:    "randomBody size for json must be at least 2",
		},
		{
			name:        "json size 2 is valid (empty object)",
			randomBody:  &RandomBodySpec{Type: "json", Size: "2"},
			expectError: false,
		},
		{
			name:        "json size 3-6 is invalid (gap between {} and smallest keyed object)",
			randomBody:  &RandomBodySpec{Type: "json", Size: "5"},
			expectError: true,
			errorMsg:    "randomBody size for json must be 2 or at least 7",
		},
		{
			name:        "json size 7 is valid (smallest keyed object)",
			randomBody:  &RandomBodySpec{Type: "json", Size: "7"},
			expectError: false,
		},
		{
			name:        "json size 8 is valid",
			randomBody:  &RandomBodySpec{Type: "json", Size: "8"},
			expectError: false,
		},
		{
			name:        "json size 9 is valid",
			randomBody:  &RandomBodySpec{Type: "json", Size: "9"},
			expectError: false,
		},
		{
			name:        "xml size too small",
			randomBody:  &RandomBodySpec{Type: "xml", Size: "6"},
			expectError: true,
			errorMsg:    "randomBody size for xml must be at least 7",
		},
		{
			name:        "xml size minimum is valid",
			randomBody:  &RandomBodySpec{Type: "xml", Size: "7"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Port: 8080},
				Requests: []RequestRule{
					{
						Path:   "/test",
						Method: "GET",
						Response: ResponseSpec{
							StatusCode: 200,
							Body:       tt.body,
							RandomBody: tt.randomBody,
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

func TestValidateRandomBody_SizeParsedIntoSizeBytes(t *testing.T) {
	tests := []struct {
		size      string
		wantBytes int
	}{
		{"512", 512},
		{"1 KB", 1024},
		{"2 MB", 2 * 1024 * 1024},
		{"1gb", 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			rb := &RandomBodySpec{Type: "plaintext", Size: tt.size}
			cfg := &Config{
				Server: ServerConfig{Port: 8080},
				Requests: []RequestRule{
					{Path: "/test", Method: "GET", Response: ResponseSpec{StatusCode: 200, RandomBody: rb}},
				},
			}
			if err := cfg.validate(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rb.SizeBytes != tt.wantBytes {
				t.Errorf("SizeBytes = %d, want %d", rb.SizeBytes, tt.wantBytes)
			}
		})
	}
}

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
