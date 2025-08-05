package config

import (
	"encoding/json"
	"fmt"
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
	RequestTimeout     int              `json:"request_timeout"`
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

	if config.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required")
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
		}
	}

	return nil
}

func setDefaults(config *Config) {
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30
	}
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