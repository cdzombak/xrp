package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"plugin"
	"reflect"
	"sync"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	"xrp/internal/config"
	xrpPlugin "xrp/pkg/xrpplugin"
)

type LoadedPlugin struct {
	plugin xrpPlugin.Plugin
	path   string
	name   string
}

func (lp *LoadedPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	if htmlPlugin, ok := lp.plugin.(xrpPlugin.HTMLPlugin); ok {
		return htmlPlugin.ProcessHTMLTree(ctx, url, node)
	}
	return lp.plugin.ProcessHTMLTree(ctx, url, node)
}

func (lp *LoadedPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	if xmlPlugin, ok := lp.plugin.(xrpPlugin.XMLPlugin); ok {
		return xmlPlugin.ProcessXMLTree(ctx, url, doc)
	}
	return lp.plugin.ProcessXMLTree(ctx, url, doc)
}

type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*LoadedPlugin
}

func New() (*Manager, error) {
	return &Manager{
		plugins: make(map[string]*LoadedPlugin),
	}, nil
}

func (m *Manager) LoadPlugins(cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newPlugins := make(map[string]*LoadedPlugin)

	for _, mimeTypeConfig := range cfg.MimeTypes {
		for _, pluginConfig := range mimeTypeConfig.Plugins {
			key := pluginConfig.Path + "/" + pluginConfig.Name

			if existing, exists := m.plugins[key]; exists {
				newPlugins[key] = existing
				continue
			}

			loadedPlugin, err := m.loadPlugin(pluginConfig.Path, pluginConfig.Name, mimeTypeConfig.MimeType)
			if err != nil {
				return fmt.Errorf("failed to load plugin %s: %w", key, err)
			}

			newPlugins[key] = loadedPlugin
			slog.Info("Loaded plugin", "path", pluginConfig.Path, "name", pluginConfig.Name)
		}
	}

	m.plugins = newPlugins
	return nil
}

func (m *Manager) loadPlugin(path, name, mimeType string) (*LoadedPlugin, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin file: %w", err)
	}

	symbol, err := p.Lookup(name)
	if err != nil {
		return nil, fmt.Errorf("failed to find symbol '%s' in plugin: %w", name, err)
	}

	// Check if the symbol implements the required methods using reflection
	// This avoids Go plugin interface identity issues
	pluginInstance, err := m.validateAndWrapPlugin(symbol, name)
	if err != nil {
		return nil, err
	}

	if err := m.validatePlugin(pluginInstance, mimeType); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %w", err)
	}

	return &LoadedPlugin{
		plugin: pluginInstance,
		path:   path,
		name:   name,
	}, nil
}

// pluginWrapper wraps a plugin symbol and implements the Plugin interface
type pluginWrapper struct {
	symbol interface{}
}

func (pw *pluginWrapper) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	method := reflect.ValueOf(pw.symbol).MethodByName("ProcessHTMLTree")
	if !method.IsValid() {
		return fmt.Errorf("ProcessHTMLTree method not found")
	}
	
	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(url),
		reflect.ValueOf(node),
	})
	
	if len(results) > 0 && !results[0].IsNil() {
		return results[0].Interface().(error)
	}
	return nil
}

func (pw *pluginWrapper) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	method := reflect.ValueOf(pw.symbol).MethodByName("ProcessXMLTree")
	if !method.IsValid() {
		return fmt.Errorf("ProcessXMLTree method not found")
	}
	
	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(url),
		reflect.ValueOf(doc),
	})
	
	if len(results) > 0 && !results[0].IsNil() {
		return results[0].Interface().(error)
	}
	return nil
}

func (m *Manager) validateAndWrapPlugin(symbol interface{}, name string) (xrpPlugin.Plugin, error) {
	// Try direct type assertion first
	if plugin, ok := symbol.(xrpPlugin.Plugin); ok {
		return plugin, nil
	}
	
	// If that fails, use reflection to check methods
	symbolValue := reflect.ValueOf(symbol)
	symbolType := reflect.TypeOf(symbol)
	
	// Handle pointer-to-pointer case (common with Go plugins)
	if symbolValue.Kind() == reflect.Ptr && symbolValue.Elem().Kind() == reflect.Ptr {
		// Dereference once to get the actual plugin instance
		symbolValue = symbolValue.Elem()
		symbol = symbolValue.Interface()
	}
	
	// Now check if it's a pointer to struct
	if symbolValue.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("symbol '%s' is not a pointer (got %v)", name, symbolValue.Kind())
	}
	
	if symbolValue.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("symbol '%s' is not a pointer to struct (got pointer to %v)", name, symbolValue.Elem().Kind())
	}
	
	slog.Info("Plugin symbol validation", "name", name, "type", symbolType, "kind", symbolValue.Kind())
	
	// Check if required methods exist
	processHTMLMethod := symbolValue.MethodByName("ProcessHTMLTree")
	processXMLMethod := symbolValue.MethodByName("ProcessXMLTree")
	
	if !processHTMLMethod.IsValid() || !processXMLMethod.IsValid() {
		return nil, fmt.Errorf("symbol '%s' does not implement required methods", name)
	}
	
	// Validate method signatures
	if err := m.validateMethodSignature(processHTMLMethod, "ProcessHTMLTree"); err != nil {
		return nil, fmt.Errorf("symbol '%s' ProcessHTMLTree method invalid: %w", name, err)
	}
	
	if err := m.validateMethodSignature(processXMLMethod, "ProcessXMLTree"); err != nil {
		return nil, fmt.Errorf("symbol '%s' ProcessXMLTree method invalid: %w", name, err)
	}
	
	return &pluginWrapper{symbol: symbol}, nil
}

func (m *Manager) validateMethodSignature(method reflect.Value, methodName string) error {
	methodType := method.Type()
	
	// Check parameter count (context, url, node/doc)
	if methodType.NumIn() != 3 {
		return fmt.Errorf("%s method should have 3 parameters", methodName)
	}
	
	// Check return count (error)
	if methodType.NumOut() != 1 {
		return fmt.Errorf("%s method should return 1 value", methodName)
	}
	
	// Check if last return type implements error interface
	returnType := methodType.Out(0)
	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	if !returnType.Implements(errorInterface) {
		return fmt.Errorf("%s method should return error", methodName)
	}
	
	return nil
}

func (m *Manager) validatePlugin(p xrpPlugin.Plugin, mimeType string) error {
	// Plugin validation passed - methods exist and have correct signatures
	// We don't call the methods with nil values as this can cause panics
	slog.Info("Plugin validation successful", "mimeType", mimeType)
	return nil
}

func (m *Manager) GetPlugin(path, name string) *LoadedPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := path + "/" + name
	return m.plugins[key]
}
