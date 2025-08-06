module minimal-plugin

go 1.21

require (
	github.com/beevik/etree v1.3.0
	github.com/cdzombak/xrp v0.0.0
	golang.org/x/net v0.19.0
)

// For local development, replace with local XRP
// replace github.com/cdzombak/xrp => ../../../..