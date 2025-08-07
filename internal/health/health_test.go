package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHealthHandler_NotReady tests health endpoint returns 102 when not ready
func TestHealthHandler_NotReady(t *testing.T) {
	server := New(8081) // port doesn't matter for handler tests

	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	server.healthHandler(recorder, req)

	if recorder.Code != http.StatusProcessing {
		t.Errorf("expected status %d, got %d", http.StatusProcessing, recorder.Code)
	}

	body := recorder.Body.String()
	if body != "starting" {
		t.Errorf("expected body 'starting', got '%s'", body)
	}
}

// TestHealthHandler_Ready tests health endpoint returns 200 when ready
func TestHealthHandler_Ready(t *testing.T) {
	server := New(8081)
	server.MarkReady()

	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	server.healthHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	if body != "ok" {
		t.Errorf("expected body 'ok', got '%s'", body)
	}

	// Check Content-Type header
	contentType := recorder.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/plain; charset=utf-8', got '%s'", contentType)
	}
}

// TestHealthHandler_MethodNotAllowed tests non-GET requests
func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	server := New(8081)

	methods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			recorder := httptest.NewRecorder()

			server.healthHandler(recorder, req)

			if recorder.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s, got %d", 
					http.StatusMethodNotAllowed, method, recorder.Code)
			}
		})
	}
}

// TestHealthServer_StateTransitions tests ready/not ready state changes
func TestHealthServer_StateTransitions(t *testing.T) {
	server := New(8081)

	// Initial state should be not ready
	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()
	server.healthHandler(recorder, req)

	if recorder.Code != http.StatusProcessing {
		t.Errorf("expected initial state to be not ready (102), got %d", recorder.Code)
	}

	// Mark ready
	server.MarkReady()
	recorder = httptest.NewRecorder()
	server.healthHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected ready state (200), got %d", recorder.Code)
	}

	// Mark not ready again
	server.MarkNotReady()
	recorder = httptest.NewRecorder()
	server.healthHandler(recorder, req)

	if recorder.Code != http.StatusProcessing {
		t.Errorf("expected not ready state (102), got %d", recorder.Code)
	}
}

// TestHealthServer_ConcurrentAccess tests thread safety of state changes
func TestHealthServer_ConcurrentAccess(t *testing.T) {
	server := New(8081)

	done := make(chan bool)
	
	// Goroutine that continuously toggles ready state
	go func() {
		for i := 0; i < 100; i++ {
			server.MarkReady()
			time.Sleep(time.Microsecond)
			server.MarkNotReady()
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine that continuously makes requests
	go func() {
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			recorder := httptest.NewRecorder()
			server.healthHandler(recorder, req)
			
			// Should get either 200 or 102, never anything else
			if recorder.Code != http.StatusOK && recorder.Code != http.StatusProcessing {
				t.Errorf("unexpected status code during concurrent access: %d", recorder.Code)
			}
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}

// TestHealthServer_Integration tests the full server lifecycle
func TestHealthServer_Integration(t *testing.T) {
	// Use port 0 to get a random available port
	server := New(0)
	
	// Start server in background
	serverChan := make(chan error, 1)
	go func() {
		serverChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Make a test request - need to discover the actual port
	// For this integration test, we'll use the handler directly
	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()
	
	server.healthHandler(recorder, req)
	
	if recorder.Code != http.StatusProcessing {
		t.Errorf("expected 102 during startup, got %d", recorder.Code)
	}

	server.MarkReady()
	
	recorder = httptest.NewRecorder()
	server.healthHandler(recorder, req)
	
	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200 when ready, got %d", recorder.Code)
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}

	// Check if server stopped (should get context deadline exceeded or similar)
	select {
	case err := <-serverChan:
		if err != http.ErrServerClosed {
			t.Logf("Server stopped with: %v (expected)", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not stop within timeout")
	}
}