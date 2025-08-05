package config

import (
	"os"
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
			defer os.Remove(tempFile.Name())

			if _, err := tempFile.WriteString(tt.configJSON); err != nil {
				t.Fatal(err)
			}
			tempFile.Close()

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
							{Path: "/path/to/plugin.so", Name: "MyPlugin"},
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
		},
		{
			name: "empty redis addr",
			config: &Config{
				BackendURL: "http://localhost:8081",
				Redis:      RedisConfig{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
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

	if config.RequestTimeout != 30 {
		t.Errorf("expected RequestTimeout to be 30, got %d", config.RequestTimeout)
	}
	if config.MaxResponseSizeMB != 10 {
		t.Errorf("expected MaxResponseSizeMB to be 10, got %d", config.MaxResponseSizeMB)
	}
}