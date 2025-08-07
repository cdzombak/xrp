package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
	}{
		{
			name: "valid config",
			configJSON: `{
				"backend_url": "http://localhost:8081",
				"redis": {
					"addr": "localhost:6379",
					"password": "",
					"db": 0
				},
				"mime_types": [
					{
						"mime_type": "text/html",
						"plugins": [
							{
								"path": "/path/to/plugin.so",
								"name": "MyPlugin"
							}
						]
					}
				],
				"cookie_denylist": ["session"]
			}`,
			expectError: false,
		},
		{
			name: "missing backend_url",
			configJSON: `{
				"redis": {
					"addr": "localhost:6379",
					"password": "",
					"db": 0
				},
				"mime_types": []
			}`,
			expectError: true,
		},
		{
			name: "invalid mime type",
			configJSON: `{
				"backend_url": "http://localhost:8081",
				"redis": {
					"addr": "localhost:6379",
					"password": "",
					"db": 0
				},
				"mime_types": [
					{
						"mime_type": "image/jpeg",
						"plugins": [
							{
								"path": "/path/to/plugin.so",
								"name": "MyPlugin"
							}
						]
					}
				]
			}`,
			expectError: true,
		},
		{
			name: "no plugins for mime type",
			configJSON: `{
				"backend_url": "http://localhost:8081",
				"redis": {
					"addr": "localhost:6379",
					"password": "",
					"db": 0
				},
				"mime_types": [
					{
						"mime_type": "text/html",
						"plugins": []
					}
				]
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "config-test-*.json")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Remove(tempFile.Name()) }()

			if _, err := tempFile.WriteString(tt.configJSON); err != nil {
				t.Fatal(err)
			}
			_ = tempFile.Close()

			config, err := Load(tempFile.Name())
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if config == nil {
					t.Error("config is nil")
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{Addr: "localhost:6379"},
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.so", Name: "MyPlugin"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty backend URL",
			config: &Config{
				Redis: RedisConfig{Addr: "localhost:6379"},
			},
			expectError: true,
			errorMsg:    "backend_url is required",
		},
		{
			name: "empty redis addr",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{},
			},
			expectError: true,
			errorMsg:    "redis.addr is required",
		},
		{
			name: "invalid backend URL",
			config: &Config{
				BackendURL: "not-a-url",
				Redis:      RedisConfig{Addr: "localhost:6379"},
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.so", Name: "MyPlugin"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "backend_url must be a valid HTTP/HTTPS URL",
		},
		{
			name: "plugin name without Plugin suffix",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{Addr: "localhost:6379"},
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.so", Name: "MyModule"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "plugin name 'MyModule' should end with 'Plugin'",
		},
		{
			name: "plugin path not .so file",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{Addr: "localhost:6379"},
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.exe", Name: "MyPlugin"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "plugin path './plugins/plugin.exe' must end with '.so'",
		},
		{
			name: "negative max response size",
			config: &Config{
				BackendURL:        "http://localhost:8081",
				Redis:             RedisConfig{Addr: "localhost:6379"},
				MaxResponseSizeMB: -1,
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.so", Name: "MyPlugin"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "max_response_size_mb must be positive",
		},
		{
			name: "negative health port",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{Addr: "localhost:6379"},
				HealthPort: -1,
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.so", Name: "MyPlugin"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "health_port must be between 0 and 65535",
		},
		{
			name: "health port too high",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{Addr: "localhost:6379"},
				HealthPort: 70000,
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.so", Name: "MyPlugin"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "health_port must be between 0 and 65535",
		},
		{
			name: "valid health port zero (random)",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{Addr: "localhost:6379"},
				HealthPort: 0,
				MimeTypes: []MimeTypeConfig{
					{
						MimeType: "text/html",
						Plugins: []PluginConfig{
							{Path: "./plugins/plugin.so", Name: "MyPlugin"},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsHTMLXMLMimeType(t *testing.T) {
	config := &Config{
		MimeTypes: []MimeTypeConfig{
			{MimeType: "text/html"},
			{MimeType: "application/xml"},
		},
	}

	tests := []struct {
		mimeType string
		expected bool
	}{
		{"text/html", true},
		{"application/xml", true},
		{"image/jpeg", false},
		{"text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := config.IsHTMLXMLMimeType(tt.mimeType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	config := &Config{}
	setDefaults(config)

	if config.MaxResponseSizeMB != 10 {
		t.Errorf("expected MaxResponseSizeMB to be 10, got %d", config.MaxResponseSizeMB)
	}

	if config.HealthPort != 8081 {
		t.Errorf("expected HealthPort to be 8081, got %d", config.HealthPort)
	}
}