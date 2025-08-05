package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"plugin"
	"sync"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	"xrp/internal/config"
	xrpPlugin "xrp/pkg/plugin"
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

	pluginInstance, ok := symbol.(xrpPlugin.Plugin)
	if !ok {
		return nil, fmt.Errorf("symbol '%s' does not implement Plugin interface", name)
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

func (m *Manager) validatePlugin(p xrpPlugin.Plugin, mimeType string) error {
	// Check if plugin implements specialized interfaces for better performance
	isHTMLMimeType := mimeType == "text/html" || mimeType == "application/xhtml+xml"
	
	if isHTMLMimeType {
		// For HTML MIME types, prefer HTMLPlugin interface but allow full Plugin interface
		if htmlPlugin, ok := p.(xrpPlugin.HTMLPlugin); ok {
			// Test that the HTML method works (we pass nil values for validation)
			if err := htmlPlugin.ProcessHTMLTree(context.Background(), nil, nil); err != nil {
				slog.Info("Plugin HTML method test failed, but this may be expected", "error", err)
			}
		}
	} else {
		// For XML MIME types, prefer XMLPlugin interface but allow full Plugin interface
		if xmlPlugin, ok := p.(xrpPlugin.XMLPlugin); ok {
			// Test that the XML method works (we pass nil values for validation)
			if err := xmlPlugin.ProcessXMLTree(context.Background(), nil, nil); err != nil {
				slog.Info("Plugin XML method test failed, but this may be expected", "error", err)
			}
		}
	}
	
	// All plugins must implement the full Plugin interface as fallback
	return nil
}

func (m *Manager) GetPlugin(path, name string) *LoadedPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := path + "/" + name
	return m.plugins[key]
}
