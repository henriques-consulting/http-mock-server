package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig  `yaml:"server"`
	Requests []RequestRule `yaml:"requests"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port uint `yaml:"port"`
}

// ResponseDelay specifies the min/max delay before sending a response
type ResponseDelay struct {
	Min int `yaml:"min"` // Minimum delay in milliseconds
	Max int `yaml:"max"` // Maximum delay in milliseconds
}

// RandomBodySpec configures pre-generated random body content for a response
type RandomBodySpec struct {
	Type      string `yaml:"type"` // "plaintext", "json", or "xml"
	Size      string `yaml:"size"` // Human-readable size, e.g. "2 MB", "512", "10kb"
	SizeBytes int    `yaml:"-"`    // Parsed from Size during config loading
}

// RequestRule defines a single mock request matching rule
type RequestRule struct {
	Path          string            `yaml:"path"`
	Headers       map[string]string `yaml:"headers"`
	QueryParams   map[string]string `yaml:"queryParams"`
	Method        string            `yaml:"method"`
	Response      ResponseSpec      `yaml:"response"`
	Body          string            `yaml:"body"`
	ResponseDelay *ResponseDelay    `yaml:"responseDelay"`
}

// ResponseSpec describes the response to return when a rule matches
type ResponseSpec struct {
	Body       interface{}       `yaml:"body"`
	RandomBody *RandomBodySpec   `yaml:"randomBody"`
	StatusCode int               `yaml:"status-code"`
	Headers    map[string]string `yaml:"headers"`
}

// Load reads and parses the configuration file
func Load() (*Config, error) {
	configPaths := []string{"config.yaml", "config/config.yaml"}

	var configData []byte
	var configPath string
	var err error

	for _, path := range configPaths {
		configData, err = os.ReadFile(path)
		if err == nil {
			configPath = path
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("could not find config file in any of %v: %w", configPaths, err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file %s: %w", configPath, err)
	}

	// Set defaults and validate
	if err := config.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Log each rule details
	log.Printf("Loaded configuration from %s with %d request rules", configPath, len(config.Requests))
	for i, rule := range config.Requests {
		bodyDesc := "none"
		if rb := rule.Response.RandomBody; rb != nil {
			bodyDesc = fmt.Sprintf("random %s (%s)", rb.Type, formatBytes(rb.SizeBytes))
		} else if rule.Response.Body != nil {
			bodyDesc = "configured"
		}
		log.Printf(
			"Rule %d: Path=%s, Method=%s, Headers=%v, QueryParams=%v, Body=%v, ResponseDelay=%v",
			i+1, rule.Path, rule.Method, rule.Headers, rule.QueryParams, bodyDesc, rule.ResponseDelay,
		)
	}
	return &config, nil
}

func (c *Config) setDefaults() error {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}

	for i := range c.Requests {
		rule := &c.Requests[i]
		if rule.Method == "" {
			rule.Method = "GET"
		}
		rule.Method = strings.ToUpper(rule.Method)

		if rule.Response.StatusCode == 0 {
			rule.Response.StatusCode = 200
		}
	}

	return nil
}

// MaxRandomBodySizeBytes is the maximum allowed random body size (2 GB) to support large payload and streaming testing.
const MaxRandomBodySizeBytes = 2 * 1024 * 1024 * 1024

// parseSize parses a human-readable size string into bytes.
// The numeric part is an integer; the optional unit suffix (case-insensitive)
// may be b, k/kb, m/mb, or g/gb. No suffix means bytes.
// Examples: "512", "1000b", "10k", "10kb", "2 MB", "1 GB"
func parseSize(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}
	if s[0] == '-' {
		return 0, fmt.Errorf("size cannot be negative")
	}

	// Split at the boundary between digits and the unit suffix.
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("size %q must start with a number", s)
	}

	n, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}

	unit := strings.ToLower(strings.TrimSpace(s[i:]))
	switch unit {
	case "", "b":
		return n, nil
	case "k", "kb":
		return n * 1024, nil
	case "m", "mb":
		return n * 1024 * 1024, nil
	case "g", "gb":
		return n * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("size %q has unknown unit %q, use b, kb, mb, or gb", s, unit)
	}
}

func formatBytes(n int) string {
	switch {
	case n >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func (c *Config) validate() error {
	if c.Server.Port == 0 {
		return fmt.Errorf("server port is required")
	}

	for i, rule := range c.Requests {
		if rule.Path == "" {
			return fmt.Errorf("request rule %d: path is required", i)
		}
		if rule.Method == "" {
			return fmt.Errorf("request rule %d: method is required", i)
		}
		if rule.Response.StatusCode < 100 || rule.Response.StatusCode > 599 {
			return fmt.Errorf("request rule %d: invalid status code %d", i, rule.Response.StatusCode)
		}
		if delay := rule.ResponseDelay; delay != nil {
			if delay.Min < 0 {
				return fmt.Errorf("request rule %d: responseDelay min cannot be negative", i)
			}
			if delay.Max < 0 {
				return fmt.Errorf("request rule %d: responseDelay max cannot be negative", i)
			}
			if delay.Min > delay.Max {
				return fmt.Errorf("request rule %d: responseDelay min (%d) cannot exceed max (%d)", i, delay.Min, delay.Max)
			}
		}
		if rb := rule.Response.RandomBody; rb != nil {
			if rule.Response.Body != nil {
				return fmt.Errorf("request rule %d: body and randomBody are mutually exclusive", i)
			}
			switch rb.Type {
			case "plaintext", "json", "xml":
				// valid
			default:
				return fmt.Errorf("request rule %d: randomBody type must be one of: plaintext, json, xml", i)
			}
			n, err := parseSize(rb.Size)
			if err != nil {
				return fmt.Errorf("request rule %d: randomBody size: %w", i, err)
			}
			rb.SizeBytes = n
			if rb.SizeBytes > MaxRandomBodySizeBytes {
				return fmt.Errorf("request rule %d: randomBody size (%s) exceeds maximum allowed (%s)", i, formatBytes(rb.SizeBytes), formatBytes(MaxRandomBodySizeBytes))
			}
			if rb.Type == "json" && rb.SizeBytes < 2 {
				return fmt.Errorf("request rule %d: randomBody size for json must be at least 2", i)
			}
			if rb.Type == "json" && rb.SizeBytes > 2 && rb.SizeBytes < 7 {
				return fmt.Errorf("request rule %d: randomBody size for json must be 2 or at least 7", i)
			}
			if rb.Type == "xml" && rb.SizeBytes < 7 {
				return fmt.Errorf("request rule %d: randomBody size for xml must be at least 7", i)
			}
		}
	}

	return nil
}
