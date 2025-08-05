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

	"golang.org/x/net/html"

	"github.com/beevik/etree"

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
	// Check content length before reading if available
	maxSize := int64(p.config.MaxResponseSizeMB * 1024 * 1024)
	if resp.ContentLength > 0 && resp.ContentLength > maxSize {
		slog.Info("Response too large, skipping processing", "size", resp.ContentLength, "max", maxSize)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close response body", "error", err)
		}
		return body, nil
	}

	// Use LimitReader to prevent reading more than maxSize
	limitedReader := io.LimitReader(resp.Body, maxSize+1) // +1 to detect if limit exceeded
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		slog.Error("Failed to close response body", "error", err)
	}

	// Check if we hit the size limit
	if int64(len(body)) > maxSize {
		slog.Info("Response too large, skipping processing", "size", len(body), "max", maxSize)
		return body, nil
	}

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
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	ctx := req.Context()
	requestURL := req.URL

	for _, pluginConfig := range pluginConfigs {
		plugin := p.plugins.GetPlugin(pluginConfig.Path, pluginConfig.Name)
		if plugin == nil {
			return nil, fmt.Errorf("plugin not found: %s/%s", pluginConfig.Path, pluginConfig.Name)
		}

		if err := plugin.ProcessHTMLTree(ctx, requestURL, doc); err != nil {
			return nil, fmt.Errorf("plugin %s failed: %w", pluginConfig.Name, err)
		}
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *Proxy) processXMLResponse(req *http.Request, body []byte, pluginConfigs []config.PluginConfig) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	ctx := req.Context()
	requestURL := req.URL

	for _, pluginConfig := range pluginConfigs {
		plugin := p.plugins.GetPlugin(pluginConfig.Path, pluginConfig.Name)
		if plugin == nil {
			return nil, fmt.Errorf("plugin not found: %s/%s", pluginConfig.Path, pluginConfig.Name)
		}

		if err := plugin.ProcessXMLTree(ctx, requestURL, doc); err != nil {
			return nil, fmt.Errorf("plugin %s failed: %w", pluginConfig.Name, err)
		}
	}

	output, err := doc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize XML: %w", err)
	}

	return output, nil
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
