package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	"xrp/pkg/xrpplugin"
)

// XMLTransformer is an example plugin that modifies XML content
type XMLTransformer struct{}

// Compile-time interface check
var _ xrpplugin.Plugin = (*XMLTransformer)(nil)

// ProcessHTMLTree is required by the interface but not used for XML
func (x *XMLTransformer) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
	return fmt.Errorf("XMLTransformer does not process HTML")
}

// ProcessXMLTree adds metadata and transforms XML content
func (x *XMLTransformer) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
	root := doc.Root()
	if root == nil {
		return fmt.Errorf("no root element found in XML document")
	}

	// Add a processing timestamp attribute to the root element
	root.CreateAttr("processed-at", time.Now().UTC().Format(time.RFC3339))
	root.CreateAttr("processed-by", "xrp-xml-transformer")

	// Add a metadata element
	metadata := root.CreateElement("metadata")
	metadata.CreateElement("processor").SetText("XRP XML Transformer")
	metadata.CreateElement("version").SetText("1.0")
	metadata.CreateElement("timestamp").SetText(time.Now().UTC().Format(time.RFC3339))

	// Transform all text content to include a prefix
	transformTextContent(root)

	return nil
}

// transformTextContent recursively processes all text elements
func transformTextContent(element *etree.Element) {
	// Process text content of current element
	if text := element.Text(); text != "" {
		element.SetText("[PROCESSED] " + text)
	}

	// Process all child elements
	for _, child := range element.ChildElements() {
		transformTextContent(child)
	}
}

// GetPlugin returns a new instance of the XML transformer plugin.
// This is the standard plugin export function that XRP will look for.
func GetPlugin() xrpplugin.Plugin {
	return &XMLTransformer{}
}
