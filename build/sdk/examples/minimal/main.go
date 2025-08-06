// Minimal XRP plugin example
// This demonstrates the basic structure of an XRP plugin

package main

import (
	"context"
	"net/url"

	"golang.org/x/net/html"
	"github.com/beevik/etree"

	"github.com/cdzombak/xrp/pkg/xrpplugin"
)

type MinimalPlugin struct{}

func (p *MinimalPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	// Example: Add a comment to the HTML
	if node.Type == html.ElementNode && node.Data == "body" {
		comment := &html.Node{
			Type: html.CommentNode,
			Data: " Modified by MinimalPlugin ",
		}
		node.AppendChild(comment)
	}
	return nil
}

func (p *MinimalPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	// Example: Add a comment to the XML
	doc.CreateComment(" Modified by MinimalPlugin ")
	return nil
}

// Export the plugin instance
func GetPlugin() xrpplugin.Plugin {
	return &MinimalPlugin{}
}