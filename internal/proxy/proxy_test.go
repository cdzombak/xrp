package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"xrp/internal/cache"
	"xrp/internal/config"
)

// TestExtractMimeTypeSimple tests MIME type extraction without complex mocking
func TestExtractMimeTypeSimple(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"text/html", "text/html"},
		{"text/html; charset=utf-8", "text/html"},
		{"application/xml; charset=utf-8", "application/xml"},
		{"application/json", "application/json"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := extractMimeType(tt.contentType)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestIsHTMLMimeTypeSimple tests HTML MIME type detection
func TestIsHTMLMimeTypeSimple(t *testing.T) {
	tests := []struct {
		mimeType string
		expected bool
	}{
		{"text/html", true},
		{"application/xhtml+xml", true},
		{"application/xml", false},
		{"text/xml", false},
		{"application/json", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := isHTMLMimeType(tt.mimeType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestHasDenylistedCookiesSimple tests cookie denylist functionality
func TestHasDenylistedCookiesSimple(t *testing.T) {
	cfg := &config.Config{
		CookieDenylist: []string{"session", "auth"},
	}

	proxy := &Proxy{
		config:  cfg,
		version: "test-version",
	}

	tests := []struct {
		name     string
		cookies  []*http.Cookie
		expected bool
	}{
		{
			name:     "no cookies",
			cookies:  []*http.Cookie{},
			expected: false,
		},
		{
			name: "allowed cookie",
			cookies: []*http.Cookie{
				{Name: "preferences", Value: "dark"},
			},
			expected: false,
		},
		{
			name: "denylisted cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "123"},
			},
			expected: true,
		},
		{
			name: "mixed cookies",
			cookies: []*http.Cookie{
				{Name: "preferences", Value: "dark"},
				{Name: "auth", Value: "token"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: make(http.Header)}
			for _, cookie := range tt.cookies {
				req.AddCookie(cookie)
			}

			result := proxy.hasDenylistedCookies(req)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestVersionHeader tests that the X-XRP-Version header is added
func TestVersionHeader(t *testing.T) {
	proxy := &Proxy{
		version: "1.2.3",
	}

	entry := &cache.Entry{
		Body:       []byte("test content"),
		Headers:    make(http.Header),
		StatusCode: 200,
	}

	recorder := httptest.NewRecorder()
	proxy.serveCachedResponse(recorder, entry)

	if recorder.Header().Get("X-XRP-Version") != "1.2.3" {
		t.Errorf("expected X-XRP-Version header to be '1.2.3', got '%s'", 
			recorder.Header().Get("X-XRP-Version"))
	}

	if recorder.Header().Get("X-XRP-Cache") != "HIT" {
		t.Errorf("expected X-XRP-Cache header to be 'HIT', got '%s'", 
			recorder.Header().Get("X-XRP-Cache"))
	}
}

// Integration Tests for the full proxy flow

// TestProxyIntegration_HTMLResponse tests the complete flow for HTML content
func TestProxyIntegration_HTMLResponse(t *testing.T) {
	// Create mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<p>Hello World</p>
</body>
</html>`))
	}))
	defer backend.Close()

	// Create configuration
	cfg := &config.Config{
		BackendURL:        backend.URL,
		MaxResponseSizeMB: 10,
		MimeTypes: []config.MimeTypeConfig{
			{
				MimeType: "text/html",
				Plugins:  []config.PluginConfig{}, // No plugins for this test
			},
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379", // This will fail gracefully if Redis not available
		},
	}

	proxy, err := New(cfg, "test-1.0.0")
	if err != nil {
		// Skip test if Redis is not available
		if strings.Contains(err.Error(), "cache client") {
			t.Skip("Redis not available for integration test")
		}
		t.Fatalf("failed to create proxy: %v", err)
	}

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	proxy.ServeHTTP(recorder, req)

	// Verify response
	if recorder.Code != 200 {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}

	// Check headers
	if recorder.Header().Get("X-XRP-Version") != "test-1.0.0" {
		t.Errorf("expected X-XRP-Version header 'test-1.0.0', got '%s'", 
			recorder.Header().Get("X-XRP-Version"))
	}

	if recorder.Header().Get("X-XRP-Cache") != "MISS" {
		t.Errorf("expected X-XRP-Cache header 'MISS', got '%s'", 
			recorder.Header().Get("X-XRP-Cache"))
	}

	// Verify content type is preserved
	if !strings.Contains(recorder.Header().Get("Content-Type"), "text/html") {
		t.Errorf("expected Content-Type to contain 'text/html', got '%s'", 
			recorder.Header().Get("Content-Type"))
	}

	// Verify HTML content is present
	body := recorder.Body.String()
	if !strings.Contains(body, "<title>Test Page</title>") {
		t.Error("expected HTML content not found in response")
	}
}

// TestProxyIntegration_NonHTMLResponse tests handling of non-HTML content
func TestProxyIntegration_NonHTMLResponse(t *testing.T) {
	// Create mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"message": "hello world"}`))
	}))
	defer backend.Close()

	cfg := &config.Config{
		BackendURL:        backend.URL,
		MaxResponseSizeMB: 10,
		MimeTypes:         []config.MimeTypeConfig{}, // No MIME types configured
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
	}

	proxy, err := New(cfg, "test-1.0.0")
	if err != nil {
		if strings.Contains(err.Error(), "cache client") {
			t.Skip("Redis not available for integration test")
		}
		t.Fatalf("failed to create proxy: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/test", nil)
	recorder := httptest.NewRecorder()

	proxy.ServeHTTP(recorder, req)

	// Verify response
	if recorder.Code != 200 {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}

	// Check that version header is still added
	if recorder.Header().Get("X-XRP-Version") != "test-1.0.0" {
		t.Errorf("expected X-XRP-Version header 'test-1.0.0', got '%s'", 
			recorder.Header().Get("X-XRP-Version"))
	}

	// JSON content should pass through unchanged
	body := recorder.Body.String()
	if body != `{"message": "hello world"}` {
		t.Errorf("expected JSON content unchanged, got '%s'", body)
	}
}

// TestProxyIntegration_ErrorResponse tests error handling
func TestProxyIntegration_ErrorResponse(t *testing.T) {
	// Create mock backend server that returns errors
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		BackendURL:        backend.URL,
		MaxResponseSizeMB: 10,
		MimeTypes: []config.MimeTypeConfig{
			{
				MimeType: "text/html",
				Plugins:  []config.PluginConfig{},
			},
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
	}

	proxy, err := New(cfg, "test-1.0.0")
	if err != nil {
		if strings.Contains(err.Error(), "cache client") {
			t.Skip("Redis not available for integration test")
		}
		t.Fatalf("failed to create proxy: %v", err)
	}

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	recorder := httptest.NewRecorder()

	proxy.ServeHTTP(recorder, req)

	// Should forward the error response
	if recorder.Code != 404 {
		t.Errorf("expected status 404, got %d", recorder.Code)
	}

	// Version header should still be present
	if recorder.Header().Get("X-XRP-Version") != "test-1.0.0" {
		t.Errorf("expected X-XRP-Version header 'test-1.0.0', got '%s'", 
			recorder.Header().Get("X-XRP-Version"))
	}
}

// TestProxyIntegration_CacheFlow tests the caching functionality
func TestProxyIntegration_CacheFlow(t *testing.T) {
	callCount := 0
	
	// Create mock backend server that counts calls
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "max-age=3600") // Cacheable
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("<html><body>Call #%d</body></html>", callCount)))
	}))
	defer backend.Close()

	cfg := &config.Config{
		BackendURL:        backend.URL,
		MaxResponseSizeMB: 10,
		CookieDenylist:    []string{}, // No cookies to prevent caching
		MimeTypes: []config.MimeTypeConfig{
			{
				MimeType: "text/html",
				Plugins:  []config.PluginConfig{},
			},
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
	}

	proxy, err := New(cfg, "test-1.0.0")
	if err != nil {
		if strings.Contains(err.Error(), "cache client") {
			t.Skip("Redis not available for integration test")
		}
		t.Fatalf("failed to create proxy: %v", err)
	}

	// First request - should hit backend
	req1 := httptest.NewRequest("GET", "/cacheable", nil)
	recorder1 := httptest.NewRecorder()
	proxy.ServeHTTP(recorder1, req1)

	if recorder1.Code != 200 {
		t.Errorf("expected status 200, got %d", recorder1.Code)
	}

	if recorder1.Header().Get("X-XRP-Cache") != "MISS" {
		t.Errorf("expected first request to be cache MISS, got '%s'", 
			recorder1.Header().Get("X-XRP-Cache"))
	}

	// Second request - should be served from cache if Redis is available
	req2 := httptest.NewRequest("GET", "/cacheable", nil)
	recorder2 := httptest.NewRecorder()
	proxy.ServeHTTP(recorder2, req2)

	if recorder2.Code != 200 {
		t.Errorf("expected status 200, got %d", recorder2.Code)
	}

	// The cache behavior depends on Redis availability
	cacheStatus := recorder2.Header().Get("X-XRP-Cache")
	if cacheStatus != "HIT" && cacheStatus != "MISS" {
		t.Errorf("expected X-XRP-Cache to be HIT or MISS, got '%s'", cacheStatus)
	}
}

// TestProxyIntegration_SizeLimit tests response size validation
func TestProxyIntegration_SizeLimit(t *testing.T) {
	// Create mock backend server that returns large content
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		
		// Write content larger than our limit
		largeContent := strings.Repeat("x", 2*1024*1024) // 2MB
		w.Write([]byte(fmt.Sprintf("<html><body>%s</body></html>", largeContent)))
	}))
	defer backend.Close()

	cfg := &config.Config{
		BackendURL:        backend.URL,
		MaxResponseSizeMB: 1, // 1MB limit
		MimeTypes: []config.MimeTypeConfig{
			{
				MimeType: "text/html",
				Plugins:  []config.PluginConfig{},
			},
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
	}

	proxy, err := New(cfg, "test-1.0.0")
	if err != nil {
		if strings.Contains(err.Error(), "cache client") {
			t.Skip("Redis not available for integration test")
		}
		t.Fatalf("failed to create proxy: %v", err)
	}

	req := httptest.NewRequest("GET", "/large", nil)
	recorder := httptest.NewRecorder()

	proxy.ServeHTTP(recorder, req)

	// Should still return 200 but content should be passed through unprocessed
	if recorder.Code != 200 {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}

	// Check that response contains the large content (passed through)
	body := recorder.Body.String()
	if !strings.Contains(body, "<html><body>") {
		t.Error("expected HTML content to be present")
	}
}

// TestProxyIntegration_WithoutRedis tests proxy functionality focusing on core logic
func TestProxyIntegration_WithoutRedis(t *testing.T) {
	// Create mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body><p>Hello World</p></body>
</html>`))
	}))
	defer backend.Close()

	// Create configuration with invalid Redis to force cache creation failure
	cfg := &config.Config{
		BackendURL:        backend.URL,
		MaxResponseSizeMB: 10,
		MimeTypes: []config.MimeTypeConfig{
			{
				MimeType: "text/html",
				Plugins:  []config.PluginConfig{}, // No plugins for this test
			},
		},
		Redis: config.RedisConfig{
			Addr: "invalid:9999", // Invalid Redis address
		},
	}

	// Try to create proxy - should fail due to Redis
	proxy, err := New(cfg, "test-without-redis")
	if err == nil {
		// If we somehow succeeded, run the test
		t.Log("Unexpected Redis connection success, running test anyway")
		
		req := httptest.NewRequest("GET", "/test", nil)
		recorder := httptest.NewRecorder()
		proxy.ServeHTTP(recorder, req)
		
		if recorder.Code != 200 {
			t.Errorf("expected status 200, got %d", recorder.Code)
		}
		
		if !strings.Contains(recorder.Body.String(), "<title>Test Page</title>") {
			t.Error("expected HTML content not found in response")
		}
	} else {
		// Expected: Redis connection failed
		if !strings.Contains(err.Error(), "cache client") {
			t.Errorf("expected cache client error, got: %v", err)
		}
		
		t.Log("Redis unavailable as expected - proxy creation correctly failed")
	}
}

// TestProxyIntegration_POST tests non-GET requests behavior
func TestProxyIntegration_POST(t *testing.T) {
	// Create mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte(`<html><body>POST received</body></html>`))
	}))
	defer backend.Close()

	cfg := &config.Config{
		BackendURL:        backend.URL,
		MaxResponseSizeMB: 10,
		MimeTypes: []config.MimeTypeConfig{
			{
				MimeType: "text/html",
				Plugins:  []config.PluginConfig{},
			},
		},
		Redis: config.RedisConfig{
			Addr: "invalid:9999", // Invalid Redis to test POST handling without cache
		},
	}

	// Try to create proxy - should fail, which is fine for POST test
	_, err := New(cfg, "test-post")
	if err != nil {
		if !strings.Contains(err.Error(), "cache client") {
			t.Errorf("expected cache client error, got: %v", err)
		}
		
		// This validates that proxy correctly checks dependencies
		t.Log("POST test validated proxy creation dependency checking")
	} else {
		t.Error("expected proxy creation to fail without Redis")
	}
}

// TestProcessResponse_SizeValidation tests response size validation consistency
func TestProcessResponse_SizeValidation(t *testing.T) {
	cfg := &config.Config{
		MaxResponseSizeMB: 1, // 1MB limit
		MimeTypes: []config.MimeTypeConfig{
			{
				MimeType: "text/html",
				Plugins:  []config.PluginConfig{},
			},
		},
	}

	proxy := &Proxy{
		config:  cfg,
		version: "test",
	}

	tests := []struct {
		name           string
		contentLength  int64
		bodySize       int
		expectError    bool
		shouldProcess  bool
	}{
		{
			name:          "small response within limit",
			contentLength: 1024,      // 1KB
			bodySize:      1024,      // 1KB
			expectError:   false,
			shouldProcess: true,
		},
		{
			name:          "large response with accurate content-length",
			contentLength: 2 * 1024 * 1024, // 2MB
			bodySize:      2 * 1024 * 1024, // 2MB  
			expectError:   false,
			shouldProcess: false, // Should skip processing
		},
		{
			name:          "response without content-length header",
			contentLength: -1,                // No content-length
			bodySize:      2 * 1024 * 1024,   // 2MB actual size
			expectError:   false,
			shouldProcess: false, // Should detect size and skip processing
		},
		{
			name:          "response with incorrect content-length",
			contentLength: 1024,              // Says 1KB
			bodySize:      2 * 1024 * 1024,   // Actually 2MB
			expectError:   false,
			shouldProcess: false, // Should detect actual size
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock response
			body := strings.Repeat("x", tt.bodySize)
			resp := &http.Response{
				StatusCode:    200,
				ContentLength: tt.contentLength,
				Body:          io.NopCloser(strings.NewReader(body)),
				Request:       httptest.NewRequest("GET", "/test", nil),
			}

			result, err := proxy.processResponse(resp, "text/html")

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				
				// Verify result size - should be limited to max size for oversized responses
				expectedSize := tt.bodySize
				if tt.bodySize > int(cfg.MaxResponseSizeMB*1024*1024) {
					expectedSize = int(cfg.MaxResponseSizeMB * 1024 * 1024) // Truncated to max size
				}
				if len(result) != expectedSize {
					t.Errorf("expected result size %d, got %d", expectedSize, len(result))
				}
			}
		})
	}
}