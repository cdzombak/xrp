// Minimal XRP plugin example
// This demonstrates the basic structure of an XRP plugin

package main

import (
	"context"
	"net/url"

	"golang.org/x/net/html"
	"github.com/beevik/etree"
)

// MinimalPlugin implements the XRP plugin interface
type MinimalPlugin struct{}

// ProcessHTMLTree processes HTML content
func (p *MinimalPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	// Example: Add a comment to the HTML body
	if node.Type == html.ElementNode && node.Data == "body" {
		comment := &html.Node{
			Type: html.CommentNode,
			Data: " Modified by MinimalPlugin ",
		}
		node.AppendChild(comment)
	}
	return nil
}

// ProcessXMLTree processes XML content  
func (p *MinimalPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	// Example: Add a comment to the XML document
	doc.CreateComment(" Modified by MinimalPlugin ")
	return nil
}

// CRITICAL: Export struct value (not pointer) to avoid plugin system issues
// The Go plugin system with reflection fallback requires this exact pattern
var MinimalPluginInstance = MinimalPlugin{}