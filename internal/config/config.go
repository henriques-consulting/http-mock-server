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

// RequestRule defines a single mock request matching rule
type RequestRule struct {
	Path     string            `yaml:"path"`
	Headers  map[string]string `yaml:"headers"`
	Method   string            `yaml:"method"`
	Response ResponseSpec      `yaml:"response"`
	Body     string            `yaml:"body"`
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
			"Rule %d: Path=%s, Method=%s, Headers=%v, Body=%v",
			i+1, rule.Path, rule.Method, rule.Headers, rule.Body,
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
	}

	return nil
}
