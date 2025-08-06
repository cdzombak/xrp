// This file contains the common plugin processing logic for XRP.
// It provides a generic framework for processing any document type (HTML, XML)
// with plugins while maintaining type safety and consistent error handling.
package proxy

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	"xrp/internal/config"
	"xrp/internal/plugins"
)

// ProcessorFunc defines a function that processes a document with a plugin
type ProcessorFunc func(plugin *plugins.LoadedPlugin, ctx context.Context, url *url.URL, document interface{}) error

// ParserFunc defines a function that parses body bytes into a document
type ParserFunc func(body []byte) (interface{}, error)

// RendererFunc defines a function that renders a document back to bytes
type RendererFunc func(document interface{}) ([]byte, error)

// processWithPlugins is a generic function that processes any document type with plugins
func (p *Proxy) processWithPlugins(
	body []byte,
	req *http.Request,
	pluginConfigs []config.PluginConfig,
	parser ParserFunc,
	processor ProcessorFunc,
	renderer RendererFunc,
) ([]byte, error) {
	// Parse the document
	document, err := parser(body)
	if err != nil {
		return nil, err
	}

	// Process with plugins
	ctx := req.Context()
	requestURL := req.URL

	for _, pluginConfig := range pluginConfigs {
		plugin := p.plugins.GetPlugin(pluginConfig.Path, pluginConfig.Name)
		if plugin == nil {
			return nil, fmt.Errorf("plugin not found: %s/%s", pluginConfig.Path, pluginConfig.Name)
		}

		if err := processor(plugin, ctx, requestURL, document); err != nil {
			return nil, fmt.Errorf("plugin %s failed: %w", pluginConfig.Name, err)
		}
	}

	// Render the document back to bytes
	return renderer(document)
}

// HTML processing functions
func parseHTML(body []byte) (interface{}, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return doc, nil
}

func processHTML(plugin *plugins.LoadedPlugin, ctx context.Context, url *url.URL, document interface{}) error {
	node, ok := document.(*html.Node)
	if !ok {
		return fmt.Errorf("invalid document type for HTML processing")
	}
	return plugin.ProcessHTMLTree(ctx, url, node)
}

func renderHTML(document interface{}) ([]byte, error) {
	node, ok := document.(*html.Node)
	if !ok {
		return nil, fmt.Errorf("invalid document type for HTML rendering")
	}
	
	var buf bytes.Buffer
	if err := html.Render(&buf, node); err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}
	return buf.Bytes(), nil
}

// XML processing functions
func parseXML(body []byte) (interface{}, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	return doc, nil
}

func processXML(plugin *plugins.LoadedPlugin, ctx context.Context, url *url.URL, document interface{}) error {
	doc, ok := document.(*etree.Document)
	if !ok {
		return fmt.Errorf("invalid document type for XML processing")
	}
	return plugin.ProcessXMLTree(ctx, url, doc)
}

func renderXML(document interface{}) ([]byte, error) {
	doc, ok := document.(*etree.Document)
	if !ok {
		return nil, fmt.Errorf("invalid document type for XML rendering")
	}
	
	output, err := doc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize XML: %w", err)
	}
	return output, nil
}