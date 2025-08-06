// +build ignore

package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	"xrp/pkg/xrpplugin"
)

// HTMLModifier is an example plugin that modifies HTML content
type HTMLModifier struct{}

// Compile-time interface check
var _ xrpplugin.Plugin = (*HTMLModifier)(nil)

// ProcessHTMLTree adds a custom header to HTML pages
func (h *HTMLModifier) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	// Find the head element
	head := findElement(node, "head")
	if head == nil {
		return fmt.Errorf("no head element found")
	}

	// Create a new meta tag
	meta := &html.Node{
		Type: html.ElementNode,
		Data: "meta",
		Attr: []html.Attribute{
			{Key: "name", Val: "processed-by"},
			{Key: "content", Val: "xrp-html-modifier"},
		},
	}

	// Add the meta tag to the head
	head.AppendChild(meta)

	// Find all paragraph elements and add a class
	addClassToParagraphs(node)

	return nil
}

// ProcessXMLTree is required by the interface but not used for HTML
func (h *HTMLModifier) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	return fmt.Errorf("HTMLModifier does not process XML")
}

// findElement recursively searches for an element with the given tag name
func findElement(node *html.Node, tagName string) *html.Node {
	if node.Type == html.ElementNode && node.Data == tagName {
		return node
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if result := findElement(child, tagName); result != nil {
			return result
		}
	}

	return nil
}

// addClassToParagraphs adds a CSS class to all paragraph elements
func addClassToParagraphs(node *html.Node) {
	if node.Type == html.ElementNode && node.Data == "p" {
		// Check if class attribute already exists
		classExists := false
		for i, attr := range node.Attr {
			if attr.Key == "class" {
				// Add to existing class
				if !strings.Contains(attr.Val, "xrp-processed") {
					node.Attr[i].Val += " xrp-processed"
				}
				classExists = true
				break
			}
		}

		// Add new class attribute if it doesn't exist
		if !classExists {
			node.Attr = append(node.Attr, html.Attribute{
				Key: "class",
				Val: "xrp-processed",
			})
		}
	}

	// Recursively process child nodes
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		addClassToParagraphs(child)
	}
}

// GetPlugin returns a new instance of the HTML modifier plugin.
// This is the standard plugin export function that XRP will look for.
func GetPlugin() xrpplugin.Plugin {
	return &HTMLModifier{}
}
