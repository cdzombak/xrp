// Package config provides configuration loading and validation for the XRP proxy.
//
// It supports JSON-based configuration files with the following features:
// - Backend URL validation (must be HTTP/HTTPS)
// - Redis connection configuration
// - MIME type and plugin mapping with validation
// - Plugin naming convention enforcement (must end with "Plugin")
// - Plugin file validation (must be .so files)
// - Cookie denylist for cache exclusion
// - Response size limits
//
// Configuration files are validated on load and can be hot-reloaded via SIGHUP signal.
// Invalid configurations are rejected while keeping the current configuration active.
//
// Example configuration:
//
//	{
//	  "backend_url": "http://localhost:8081",
//	  "redis": {
//	    "addr": "localhost:6379",
//	    "password": "",
//	    "db": 0
//	  },
//	  "mime_types": [
//	    {
//	      "mime_type": "text/html",
//	      "plugins": [
//	        {
//	          "path": "./plugins/html_modifier.so",
//	          "name": "HTMLModifierPlugin"
//	        }
//	      ]
//	    }
//	  ],
//	  "cookie_denylist": ["session"],
//	  "max_response_size_mb": 10
//	}
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
)

var validHTMLXMLMimeTypes = []string{
	"text/html",
	"application/xhtml+xml",
	"text/xml",
	"application/xml",
	"application/rss+xml",
	"application/atom+xml",
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type PluginConfig struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type MimeTypeConfig struct {
	MimeType string         `json:"mime_type"`
	Plugins  []PluginConfig `json:"plugins"`
}

type Config struct {
	BackendURL         string           `json:"backend_url"`
	Redis              RedisConfig      `json:"redis"`
	MimeTypes          []MimeTypeConfig `json:"mime_types"`
	CookieDenylist     []string         `json:"cookie_denylist"`
	MaxResponseSizeMB  int              `json:"max_response_size_mb"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	setDefaults(&config)

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.BackendURL == "" {
		return fmt.Errorf("backend_url is required")
	}

	// Validate backend URL format
	if _, err := url.Parse(config.BackendURL); err != nil {
		return fmt.Errorf("backend_url must be a valid HTTP/HTTPS URL: %w", err)
	}
	parsedURL, _ := url.Parse(config.BackendURL)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("backend_url must be a valid HTTP/HTTPS URL")
	}

	if config.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required")
	}

	// Validate size limits
	if config.MaxResponseSizeMB < 0 {
		return fmt.Errorf("max_response_size_mb must be positive")
	}

	for i, mimeConfig := range config.MimeTypes {
		if !slices.Contains(validHTMLXMLMimeTypes, mimeConfig.MimeType) {
			return fmt.Errorf("mime_types[%d]: invalid MIME type '%s', must be one of: %s",
				i, mimeConfig.MimeType, strings.Join(validHTMLXMLMimeTypes, ", "))
		}

		if len(mimeConfig.Plugins) == 0 {
			return fmt.Errorf("mime_types[%d]: at least one plugin must be specified", i)
		}

		for j, plugin := range mimeConfig.Plugins {
			if plugin.Path == "" {
				return fmt.Errorf("mime_types[%d].plugins[%d]: path is required", i, j)
			}
			if plugin.Name == "" {
				return fmt.Errorf("mime_types[%d].plugins[%d]: name is required", i, j)
			}

			// Validate plugin naming convention
			if !strings.HasSuffix(plugin.Name, "Plugin") {
				return fmt.Errorf("mime_types[%d].plugins[%d]: plugin name '%s' should end with 'Plugin'", i, j, plugin.Name)
			}

			// Validate plugin file extension  
			if !strings.HasSuffix(plugin.Path, ".so") {
				return fmt.Errorf("mime_types[%d].plugins[%d]: plugin path '%s' must end with '.so'", i, j, plugin.Path)
			}
		}
	}

	return nil
}

func setDefaults(config *Config) {
	if config.MaxResponseSizeMB == 0 {
		config.MaxResponseSizeMB = 10
	}
}

func (c *Config) IsHTMLXMLMimeType(mimeType string) bool {
	for _, mt := range c.MimeTypes {
		if mt.MimeType == mimeType {
			return true
		}
	}
	return false
}

func (c *Config) GetPluginsForMimeType(mimeType string) []PluginConfig {
	for _, mt := range c.MimeTypes {
		if mt.MimeType == mimeType {
			return mt.Plugins
		}
	}
	return nil
}