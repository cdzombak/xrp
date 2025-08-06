package plugins

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	xrpPlugin "xrp/pkg/xrpplugin"
)

// Mock plugin implementations for testing
type MockHTMLPlugin struct{}

func (m *MockHTMLPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	return nil
}

func (m *MockHTMLPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	return nil // Not implemented for HTML-only plugin
}

type MockXMLPlugin struct{}

func (m *MockXMLPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	return nil // Not implemented for XML-only plugin
}

func (m *MockXMLPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	return nil
}

type MockFullPlugin struct{}

func (m *MockFullPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	return nil
}

func (m *MockFullPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	return nil
}

func TestValidatePlugin(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		name        string
		plugin      xrpPlugin.Plugin
		mimeType    string
		expectError bool
	}{
		{
			name:        "full plugin with HTML mime type",
			plugin:      &MockFullPlugin{},
			mimeType:    "text/html",
			expectError: false,
		},
		{
			name:        "full plugin with XML mime type",
			plugin:      &MockFullPlugin{},
			mimeType:    "application/xml",
			expectError: false,
		},
		{
			name:        "HTML plugin with HTML mime type",
			plugin:      &MockHTMLPlugin{},
			mimeType:    "text/html",
			expectError: false,
		},
		{
			name:        "XML plugin with XML mime type",
			plugin:      &MockXMLPlugin{},
			mimeType:    "application/xml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validatePlugin(tt.plugin, tt.mimeType)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	manager, err := New()
	if err != nil {
		t.Errorf("unexpected error creating manager: %v", err)
	}
	if manager == nil {
		t.Error("manager is nil")
		return
	}
	if manager.plugins == nil {
		t.Error("plugins map is nil")
	}
}

func TestGetPlugin(t *testing.T) {
	manager := &Manager{
		plugins: make(map[string]*LoadedPlugin),
	}

	// Test getting non-existent plugin
	plugin := manager.GetPlugin("/path/to/plugin.so", "NonExistent")
	if plugin != nil {
		t.Error("expected nil for non-existent plugin")
	}

	// Add a mock plugin directly to test retrieval
	mockPlugin := &LoadedPlugin{
		plugin: &MockFullPlugin{},
		path:   "/path/to/plugin.so",
		name:   "TestPlugin",
	}
	manager.plugins["/path/to/plugin.so/TestPlugin"] = mockPlugin

	// Test getting existing plugin
	plugin = manager.GetPlugin("/path/to/plugin.so", "TestPlugin")
	if plugin == nil {
		t.Error("expected plugin but got nil")
	}
	if plugin != mockPlugin {
		t.Error("got different plugin than expected")
	}
}

func TestLoadedPluginMethods(t *testing.T) {
	tests := []struct {
		name        string
		plugin      xrpPlugin.Plugin
		testHTML    bool
		testXML     bool
		expectError bool
	}{
		{
			name:        "full plugin - HTML",
			plugin:      &MockFullPlugin{},
			testHTML:    true,
			expectError: false,
		},
		{
			name:        "full plugin - XML",
			plugin:      &MockFullPlugin{},
			testXML:     true,
			expectError: false,
		},
		{
			name:        "HTML plugin - HTML",
			plugin:      &MockHTMLPlugin{},
			testHTML:    true,
			expectError: false,
		},
		{
			name:        "XML plugin - XML",
			plugin:      &MockXMLPlugin{},
			testXML:     true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadedPlugin := &LoadedPlugin{
				plugin: tt.plugin,
				name:   "TestPlugin",
			}

			if tt.testHTML {
				err := loadedPlugin.ProcessHTMLTree(context.Background(), nil, nil)
				if tt.expectError && err == nil {
					t.Error("expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.testXML {
				err := loadedPlugin.ProcessXMLTree(context.Background(), nil, nil)
				if tt.expectError && err == nil {
					t.Error("expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Mock plugin that captures the URL for testing
type URLCapturingPlugin struct {
	CapturedURL *url.URL
}

func (u *URLCapturingPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	u.CapturedURL = url
	return nil
}

func (u *URLCapturingPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	u.CapturedURL = url
	return nil
}

func TestPluginReceivesURL(t *testing.T) {
	testURL := &url.URL{
		Scheme: "https",
		Host:   "example.com",
		Path:   "/test/path",
	}

	urlCapturingPlugin := &URLCapturingPlugin{}
	loadedPlugin := &LoadedPlugin{
		plugin: urlCapturingPlugin,
		name:   "URLTestPlugin",
	}

	// Test HTML processing
	err := loadedPlugin.ProcessHTMLTree(context.Background(), testURL, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if urlCapturingPlugin.CapturedURL == nil {
		t.Error("URL was not passed to HTML plugin")
	} else if urlCapturingPlugin.CapturedURL.String() != testURL.String() {
		t.Errorf("expected URL %s, got %s", testURL.String(), urlCapturingPlugin.CapturedURL.String())
	}

	// Reset for XML test
	urlCapturingPlugin.CapturedURL = nil

	// Test XML processing
	err = loadedPlugin.ProcessXMLTree(context.Background(), testURL, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if urlCapturingPlugin.CapturedURL == nil {
		t.Error("URL was not passed to XML plugin")
	} else if urlCapturingPlugin.CapturedURL.String() != testURL.String() {
		t.Errorf("expected URL %s, got %s", testURL.String(), urlCapturingPlugin.CapturedURL.String())
	}
}

func TestValidatePluginSecurity(t *testing.T) {
	manager := &Manager{}

	// Create a plugins directory in temp for testing
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory to make relative paths work
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		setupFile   func() string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid plugin file",
			setupFile: func() string {
				path := filepath.Join("plugins", "valid_plugin.so")
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				file.Close()
				
				// Set proper permissions (not world-writable)
				if err := os.Chmod(path, 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			expectError: false,
		},
		{
			name: "world-writable plugin file",
			setupFile: func() string {
				path := filepath.Join("plugins", "writable_plugin.so")
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				file.Close()
				
				// Set world-writable permissions
				if err := os.Chmod(path, 0666); err != nil {
					t.Fatal(err)
				}
				return path
			},
			expectError: true,
			errorMsg:    "world-writable",
		},
		{
			name: "symlink to plugin file",
			setupFile: func() string {
				// Create target file with absolute path
				targetPath := filepath.Join("plugins", "target_plugin.so")
				file, err := os.Create(targetPath)
				if err != nil {
					t.Fatal(err)
				}
				file.Close()
				
				// Create symlink with absolute target path
				symlinkPath := filepath.Join("plugins", "symlink_plugin.so")
				absTargetPath, _ := filepath.Abs(targetPath)
				if err := os.Symlink(absTargetPath, symlinkPath); err != nil {
					t.Skip("Cannot create symlink for test")
				}
				return symlinkPath
			},
			expectError: true,
			errorMsg:    "cannot be a symlink",
		},
		{
			name: "nonexistent plugin file",
			setupFile: func() string {
				return filepath.Join("plugins", "nonexistent.so")
			},
			expectError: true,
			errorMsg:    "no such file",
		},
		{
			name: "plugin outside allowed directory",
			setupFile: func() string {
				// Create file in /tmp (outside allowed dirs)
				path := filepath.Join("/tmp", "outside_plugin.so")
				file, err := os.Create(path)
				if err != nil {
					t.Skip("cannot create temp file for test")
				}
				file.Close()
				t.Cleanup(func() {
					os.Remove(path)
				})
				return path
			},
			expectError: true,
			errorMsg:    "not in allowed directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pluginPath := tt.setupFile()
			
			err := manager.validatePluginSecurity(pluginPath)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !containsIgnoreCase(err.Error(), tt.errorMsg) {
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

// Helper function for case-insensitive string matching
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
