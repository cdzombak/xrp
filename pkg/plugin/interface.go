package plugin

import (
	"golang.org/x/net/html"

	"github.com/beevik/etree"
)

// Plugin defines the interface that all XRP plugins must implement.
// Plugins should implement either ProcessHTMLTree or ProcessXMLTree
// depending on the MIME types they are configured to handle.
type Plugin interface {
	// ProcessHTMLTree modifies an HTML tree in place.
	// It should return an error if processing fails.
	ProcessHTMLTree(node *html.Node) error

	// ProcessXMLTree modifies an XML tree in place.
	// It should return an error if processing fails.
	ProcessXMLTree(doc *etree.Document) error
}

// HTMLPlugin is a convenience interface for plugins that only handle HTML.
type HTMLPlugin interface {
	ProcessHTMLTree(node *html.Node) error
}

// XMLPlugin is a convenience interface for plugins that only handle XML.
type XMLPlugin interface {
	ProcessXMLTree(doc *etree.Document) error
}