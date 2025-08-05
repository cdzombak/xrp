package plugins

import (
	"fmt"
	"log/slog"
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

func (lp *LoadedPlugin) ProcessHTMLTree(node *html.Node) error {
	if htmlPlugin, ok := lp.plugin.(xrpPlugin.HTMLPlugin); ok {
		return htmlPlugin.ProcessHTMLTree(node)
	}
	return lp.plugin.ProcessHTMLTree(node)
}

func (lp *LoadedPlugin) ProcessXMLTree(doc *etree.Document) error {
	if xmlPlugin, ok := lp.plugin.(xrpPlugin.XMLPlugin); ok {
		return xmlPlugin.ProcessXMLTree(doc)
	}
	return lp.plugin.ProcessXMLTree(doc)
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
	// All plugins must implement the full Plugin interface, so no validation needed
	_ = p      // Use the parameter to avoid unused parameter warning
	_ = mimeType
	return nil
}

func (m *Manager) GetPlugin(path, name string) *LoadedPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := path + "/" + name
	return m.plugins[key]
}