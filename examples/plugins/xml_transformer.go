package main

import (
	"fmt"
	"time"

	"golang.org/x/net/html"

	"github.com/beevik/etree"

	"xrp/pkg/plugin"
)

// XMLTransformer is an example plugin that modifies XML content
type XMLTransformer struct{}

// ProcessHTMLTree is required by the interface but not used for XML
func (x *XMLTransformer) ProcessHTMLTree(node *html.Node) error {
	return fmt.Errorf("XMLTransformer does not process HTML")
}

// ProcessXMLTree adds metadata and transforms XML content
func (x *XMLTransformer) ProcessXMLTree(doc *etree.Document) error {
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

// XMLTransformerPlugin is the plugin symbol that will be looked up
var XMLTransformerPlugin plugin.Plugin = &XMLTransformer{}