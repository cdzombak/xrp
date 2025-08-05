package plugins

import (
	"testing"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	xrpPlugin "xrp/pkg/plugin"
)

// Mock plugin implementations for testing
type MockHTMLPlugin struct{}

func (m *MockHTMLPlugin) ProcessHTMLTree(node *html.Node) error {
	return nil
}

func (m *MockHTMLPlugin) ProcessXMLTree(doc *etree.Document) error {
	return nil // Not implemented for HTML-only plugin
}

type MockXMLPlugin struct{}

func (m *MockXMLPlugin) ProcessHTMLTree(node *html.Node) error {
	return nil // Not implemented for XML-only plugin
}

func (m *MockXMLPlugin) ProcessXMLTree(doc *etree.Document) error {
	return nil
}

type MockFullPlugin struct{}

func (m *MockFullPlugin) ProcessHTMLTree(node *html.Node) error {
	return nil
}

func (m *MockFullPlugin) ProcessXMLTree(doc *etree.Document) error {
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
				err := loadedPlugin.ProcessHTMLTree(nil)
				if tt.expectError && err == nil {
					t.Error("expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.testXML {
				err := loadedPlugin.ProcessXMLTree(nil)
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