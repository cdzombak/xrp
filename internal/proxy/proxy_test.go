package proxy

import (
	"net/http"
	"net/http/httptest"
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