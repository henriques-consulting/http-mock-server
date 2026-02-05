package config

import (
	"fmt"
	"log"
	"os"
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
		log.Printf(
			"Rule %d: Path=%s, Method=%s, Headers=%v, QueryParams=%v, Body=%v, ResponseDelay=%v",
			i+1, rule.Path, rule.Method, rule.Headers, rule.QueryParams, rule.Body, rule.ResponseDelay,
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

// MaxResponseDelayMs is the maximum allowed response delay to prevent misconfiguration.
// This value must be less than the server's WriteTimeout (15 seconds) to avoid connection resets.
const MaxResponseDelayMs = 10000

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
			if delay.Max > MaxResponseDelayMs {
				return fmt.Errorf("request rule %d: responseDelay max (%d) exceeds maximum allowed (%d)", i, delay.Max, MaxResponseDelayMs)
			}
		}
	}

	return nil
}
