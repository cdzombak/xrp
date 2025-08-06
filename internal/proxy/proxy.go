// Package proxy implements the core HTTP reverse proxy functionality for XRP.
//
// This package provides an HTTP-aware reverse proxy that can intercept and modify
// HTML and XML responses using a plugin system. The proxy supports:
//
// - Intelligent Redis-based caching with HTTP compliance
// - Plugin-based content modification for HTML/XML responses  
// - Request/response size validation and security controls
// - Version headers and cache status reporting
// - Configuration hot-reloading and graceful error handling
//
// The proxy works by intercepting HTTP responses, checking if they contain
// HTML or XML content that should be processed, parsing the content into
// a document tree, running configured plugins against the tree, and then
// serializing the modified content back to the response.
//
// Example usage:
//
//	proxy, err := proxy.New(config, "v1.0.0")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	
//	http.ListenAndServe(":8080", proxy)
//
// The proxy automatically adds X-XRP-Version and X-XRP-Cache headers to
// all responses to indicate processing status and enable monitoring.
package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"xrp/internal/cache"
	"xrp/internal/config"
	"xrp/internal/plugins"
)

type Proxy struct {
	mu       sync.RWMutex
	config   *config.Config
	reverseProxy *httputil.ReverseProxy
	cache    *cache.Cache
	plugins  *plugins.Manager
	version  string
}

func New(cfg *config.Config, version string) (*Proxy, error) {
	target, err := url.Parse(cfg.BackendURL)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL: %w", err)
	}

	cacheClient, err := cache.New(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache client: %w", err)
	}

	pluginManager, err := plugins.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin manager: %w", err)
	}

	if err := pluginManager.LoadPlugins(cfg); err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	
	p := &Proxy{
		config:   cfg,
		reverseProxy: rp,
		cache:    cacheClient,
		plugins:  pluginManager,
		version:  version,
	}

	rp.ModifyResponse = p.modifyResponse

	return p, nil
}

func (p *Proxy) UpdateConfig(cfg *config.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	target, err := url.Parse(cfg.BackendURL)
	if err != nil {
		return fmt.Errorf("invalid backend URL: %w", err)
	}

	// Update cache client if Redis configuration changed
	if p.config.Redis != cfg.Redis {
		newCache, err := cache.New(cfg.Redis)
		if err != nil {
			return fmt.Errorf("failed to create new cache client: %w", err)
		}
		p.cache = newCache
	}

	if err := p.plugins.LoadPlugins(cfg); err != nil {
		return fmt.Errorf("failed to reload plugins: %w", err)
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ModifyResponse = p.modifyResponse

	p.config = cfg
	p.reverseProxy = rp

	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if r.Method == http.MethodGet {
		if cached := p.cache.Get(r, p.config); cached != nil {
			slog.Info("Serving cached response", "url", r.URL.Path)
			p.serveCachedResponse(w, cached)
			return
		}
	}

	p.reverseProxy.ServeHTTP(w, r)
}

func (p *Proxy) modifyResponse(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	mimeType := extractMimeType(contentType)

	// Always add version header to any response that goes through XRP
	resp.Header.Set("X-XRP-Version", p.version)

	if !p.config.IsHTMLXMLMimeType(mimeType) {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Add cache MISS header for processed responses
	resp.Header.Set("X-XRP-Cache", "MISS")

	var body []byte
	var err error
	
	if resp.Request.Method == http.MethodGet && p.shouldCache(resp) {
		body, err = p.processAndCacheResponse(resp, mimeType)
	} else {
		body, err = p.processResponse(resp, mimeType)
	}
	
	if err != nil {
		slog.Error("Failed to process response", "error", err)
		return err
	}
	
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))

	return nil
}

func (p *Proxy) processResponse(resp *http.Response, mimeType string) ([]byte, error) {
	maxSize := int64(p.config.MaxResponseSizeMB * 1024 * 1024)
	
	// Always use LimitedReader to prevent reading more than allowed
	// This provides consistent behavior regardless of Content-Length header accuracy  
	limitedReader := &io.LimitedReader{
		R: resp.Body,
		N: maxSize + 1, // +1 to detect if limit exceeded
	}
	
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	if err := resp.Body.Close(); err != nil {
		slog.Error("Failed to close response body", "error", err)
	}

	// Check if we hit the size limit
	actualSize := int64(len(body))
	if actualSize > maxSize {
		slog.Info("Response exceeds size limit, skipping plugin processing", 
			"size", actualSize, "max", maxSize, "content_length", resp.ContentLength)
		// Return truncated body - proxy will pass it through unchanged
		return body[:maxSize], nil
	}

	// Response is within size limits, proceed with plugin processing
	pluginConfigs := p.config.GetPluginsForMimeType(mimeType)
	if len(pluginConfigs) == 0 {
		return body, nil
	}

	if isHTMLMimeType(mimeType) {
		return p.processHTMLResponse(resp.Request, body, pluginConfigs)
	} else {
		return p.processXMLResponse(resp.Request, body, pluginConfigs)
	}
}

func (p *Proxy) processAndCacheResponse(resp *http.Response, mimeType string) ([]byte, error) {
	processedBody, err := p.processResponse(resp, mimeType)
	if err != nil {
		return nil, err
	}

	cacheEntry := &cache.Entry{
		Body:       processedBody,
		Headers:    resp.Header,
		StatusCode: resp.StatusCode,
		Timestamp:  time.Now(),
	}

	if err := p.cache.Set(resp.Request, cacheEntry, p.config); err != nil {
		slog.Error("Failed to cache response", "error", err)
	}

	return processedBody, nil
}

func (p *Proxy) processHTMLResponse(req *http.Request, body []byte, pluginConfigs []config.PluginConfig) ([]byte, error) {
	return p.processWithPlugins(body, req, pluginConfigs, parseHTML, processHTML, renderHTML)
}

func (p *Proxy) processXMLResponse(req *http.Request, body []byte, pluginConfigs []config.PluginConfig) ([]byte, error) {
	return p.processWithPlugins(body, req, pluginConfigs, parseXML, processXML, renderXML)
}

func (p *Proxy) shouldCache(resp *http.Response) bool {
	if resp.Header.Get("Set-Cookie") != "" {
		return false
	}

	if p.hasDenylistedCookies(resp.Request) {
		return false
	}

	return p.cache.IsCacheable(resp)
}

func (p *Proxy) hasDenylistedCookies(req *http.Request) bool {
	for _, denyName := range p.config.CookieDenylist {
		for _, cookie := range req.Cookies() {
			if cookie.Name == denyName {
				return true
			}
		}
	}
	return false
}

func (p *Proxy) serveCachedResponse(w http.ResponseWriter, entry *cache.Entry) {
	for key, values := range entry.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	
	// Update Content-Length to match the actual cached body length
	w.Header().Set("Content-Length", strconv.Itoa(len(entry.Body)))
	
	// Add XRP headers for cached responses
	w.Header().Set("X-XRP-Version", p.version)
	w.Header().Set("X-XRP-Cache", "HIT")
	
	w.WriteHeader(entry.StatusCode)
	if _, err := w.Write(entry.Body); err != nil {
		slog.Error("Failed to write cached response body", "error", err)
	}
}

func extractMimeType(contentType string) string {
	if idx := strings.Index(contentType, ";"); idx != -1 {
		return strings.TrimSpace(contentType[:idx])
	}
	return strings.TrimSpace(contentType)
}

func isHTMLMimeType(mimeType string) bool {
	return mimeType == "text/html" || mimeType == "application/xhtml+xml"
}
