package cache

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"xrp/internal/config"
)

func TestGenerateKey(t *testing.T) {
	cache := &Cache{}

	tests := []struct {
		name     string
		path     string
		query    string
		vary     string
		expected bool // whether keys should be different
	}{
		{
			name:     "same path and query",
			path:     "/test",
			query:    "param=value",
			expected: false,
		},
		{
			name:     "different path",
			path:     "/different",
			query:    "param=value",
			expected: true,
		},
		{
			name:     "different query",
			path:     "/test",
			query:    "param=different",
			expected: true,
		},
	}

	baseReq := &http.Request{
		URL: &url.URL{Path: "/test", RawQuery: "param=value"},
		Header: make(http.Header),
	}
	baseKey := cache.generateKey(baseReq)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{Path: tt.path, RawQuery: tt.query},
				Header: make(http.Header),
			}
			if tt.vary != "" {
				req.Header.Set("Vary", tt.vary)
			}
			
			key := cache.generateKey(req)
			
			if tt.expected && key == baseKey {
				t.Error("expected different keys but got same")
			}
			if !tt.expected && key != baseKey {
				t.Error("expected same keys but got different")
			}
		})
	}
}

func TestIsCacheable(t *testing.T) {
	cache := &Cache{}

	tests := []struct {
		name       string
		statusCode int
		method     string
		headers    map[string]string
		expected   bool
	}{
		{
			name:       "cacheable GET 200",
			statusCode: 200,
			method:     "GET",
			headers:    map[string]string{},
			expected:   true,
		},
		{
			name:       "non-200 status",
			statusCode: 404,
			method:     "GET",
			headers:    map[string]string{},
			expected:   false,
		},
		{
			name:       "POST method",
			statusCode: 200,
			method:     "POST",
			headers:    map[string]string{},
			expected:   false,
		},
		{
			name:       "no-cache header",
			statusCode: 200,
			method:     "GET",
			headers:    map[string]string{"Cache-Control": "no-cache"},
			expected:   false,
		},
		{
			name:       "no-store header",
			statusCode: 200,
			method:     "GET",
			headers:    map[string]string{"Cache-Control": "no-store"},
			expected:   false,
		},
		{
			name:       "private header",
			statusCode: 200,
			method:     "GET",
			headers:    map[string]string{"Cache-Control": "private"},
			expected:   false,
		},
		{
			name:       "set-cookie header",
			statusCode: 200,
			method:     "GET",
			headers:    map[string]string{"Set-Cookie": "session=123"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     make(http.Header),
				Request: &http.Request{
					Method: tt.method,
				},
			}

			for key, value := range tt.headers {
				resp.Header.Set(key, value)
			}

			result := cache.IsCacheable(resp)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	cache := &Cache{}
	now := time.Now()

	tests := []struct {
		name     string
		entry    *Entry
		expected bool
	}{
		{
			name: "not expired - within max age",
			entry: &Entry{
				Timestamp: now.Add(-30 * time.Minute),
				MaxAge:    func() *int { v := 3600; return &v }(), // 1 hour
			},
			expected: false,
		},
		{
			name: "expired - beyond max age",
			entry: &Entry{
				Timestamp: now.Add(-2 * time.Hour),
				MaxAge:    func() *int { v := 3600; return &v }(), // 1 hour
			},
			expected: true,
		},
		{
			name: "not expired - within expires time",
			entry: &Entry{
				Timestamp: now.Add(-30 * time.Minute),
				Expires:   func() *time.Time { t := now.Add(30 * time.Minute); return &t }(),
			},
			expected: false,
		},
		{
			name: "expired - beyond expires time",
			entry: &Entry{
				Timestamp: now.Add(-2 * time.Hour),
				Expires:   func() *time.Time { t := now.Add(-30 * time.Minute); return &t }(),
			},
			expected: true,
		},
		{
			name: "expired - default TTL exceeded",
			entry: &Entry{
				Timestamp: now.Add(-2 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.isExpired(tt.entry)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculateTTL(t *testing.T) {
	cache := &Cache{}
	now := time.Now()

	tests := []struct {
		name     string
		entry    *Entry
		expected time.Duration
	}{
		{
			name: "max age TTL",
			entry: &Entry{
				Timestamp: now,
				MaxAge:    func() *int { v := 3600; return &v }(), // 1 hour
			},
			expected: time.Hour,
		},
		{
			name: "expires TTL",
			entry: &Entry{
				Timestamp: now,
				Expires:   func() *time.Time { t := now.Add(30 * time.Minute); return &t }(),
			},
			expected: 30 * time.Minute,
		},
		{
			name: "default TTL",
			entry: &Entry{
				Timestamp: now,
			},
			expected: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.calculateTTL(tt.entry)
			
			// Allow for small time differences due to test execution time
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("expected TTL around %v, got %v (diff: %v)", tt.expected, result, diff)
			}
		})
	}
}

func TestParseMaxAge(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl string
		expected     *int
	}{
		{
			name:         "valid max-age",
			cacheControl: "max-age=3600",
			expected:     func() *int { v := 3600; return &v }(),
		},
		{
			name:         "max-age with other directives",
			cacheControl: "public, max-age=7200, must-revalidate",
			expected:     func() *int { v := 7200; return &v }(),
		},
		{
			name:         "no max-age",
			cacheControl: "public, must-revalidate",
			expected:     nil,
		},
		{
			name:         "invalid max-age",
			cacheControl: "max-age=invalid",
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMaxAge(tt.cacheControl)
			if tt.expected == nil && result != nil {
				t.Errorf("expected nil, got %v", *result)
			}
			if tt.expected != nil && result == nil {
				t.Errorf("expected %v, got nil", *tt.expected)
			}
			if tt.expected != nil && result != nil && *tt.expected != *result {
				t.Errorf("expected %v, got %v", *tt.expected, *result)
			}
		})
	}
}

func TestParseExpires(t *testing.T) {
	tests := []struct {
		name    string
		expires string
		isNil   bool
	}{
		{
			name:    "valid expires",
			expires: "Wed, 21 Oct 2015 07:28:00 GMT",
			isNil:   false,
		},
		{
			name:    "empty expires",
			expires: "",
			isNil:   true,
		},
		{
			name:    "invalid expires",
			expires: "invalid-date",
			isNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseExpires(tt.expires)
			if tt.isNil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if !tt.isNil && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// Integration test with Redis (requires Redis to be running)
func TestCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	redisConfig := config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use a different DB for testing
	}

	cache, err := New(redisConfig)
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	// Clean up test data
	defer func() {
		ctx := context.Background()
		cache.client.FlushDB(ctx)
	}()

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test", RawQuery: "param=value"},
		Header: make(http.Header),
	}

	entry := &Entry{
		Body:       []byte("test content"),
		Headers:    make(http.Header),
		StatusCode: 200,
		Timestamp:  time.Now(),
	}
	entry.Headers.Set("Content-Type", "text/html")

	cfg := &config.Config{}

	// Test Set and Get
	err = cache.Set(req, entry, cfg)
	if err != nil {
		t.Fatalf("failed to set cache entry: %v", err)
	}

	retrieved := cache.Get(req, cfg)
	if retrieved == nil {
		t.Fatal("failed to retrieve cache entry")
	}

	if string(retrieved.Body) != string(entry.Body) {
		t.Errorf("expected body %s, got %s", entry.Body, retrieved.Body)
	}

	if retrieved.StatusCode != entry.StatusCode {
		t.Errorf("expected status %d, got %d", entry.StatusCode, retrieved.StatusCode)
	}
}