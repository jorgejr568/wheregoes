package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHttpFetcherClient_Fetch(t *testing.T) {
	tests := []struct {
		name             string
		serverResponse   func(w http.ResponseWriter, r *http.Request)
		expectedStatus   int
		expectError      bool
		checkHeaders     bool
		expectedLocation string
	}{
		{
			name: "successful request with 200 status",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Hello World"))
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
			checkHeaders:   true,
		},
		{
			name: "redirect response with location header",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "https://example.com/redirected")
				w.WriteHeader(http.StatusFound)
			},
			expectedStatus:   http.StatusFound,
			expectError:      false,
			checkHeaders:     true,
			expectedLocation: "https://example.com/redirected",
		},
		{
			name: "404 not found",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectError:    false,
		},
		{
			name: "500 internal server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    false,
		},
		{
			name: "request error - invalid URL",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// This won't be called since we'll use an invalid URL
			},
			expectedStatus: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			if tt.name == "request error - invalid URL" {
				url = "invalid://url"
			} else {
				server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
				defer server.Close()
				url = server.URL
			}

			client := NewHttpFetcherClient()
			ctx := context.Background()

			response, err := client.Fetch(ctx, url)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if response.StatusCode != tt.expectedStatus {
					t.Errorf("expected status code %d, got %d", tt.expectedStatus, response.StatusCode)
				}

				if tt.checkHeaders && response.Headers == nil {
					t.Errorf("expected headers to be present")
				}

				if tt.expectedLocation != "" {
					location := response.Headers.Get("Location")
					if location != tt.expectedLocation {
						t.Errorf("expected Location header %s, got %s", tt.expectedLocation, location)
					}
				}
			}
		})
	}
}

func TestHttpFetcherClient_FetchWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHttpFetcherClient()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Fetch(ctx, server.URL)
	if err == nil {
		t.Errorf("expected timeout error but got none")
	}
}

func TestHttpFetcherClient_RequestHeaders(t *testing.T) {
	var receivedUserAgent, receivedAccept string
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		receivedAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHttpFetcherClient()
	ctx := context.Background()

	_, err := client.Fetch(ctx, server.URL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedUserAgent != "wheregoes" {
		t.Errorf("expected User-Agent 'wheregoes', got '%s'", receivedUserAgent)
	}

	if receivedAccept != "*/*" {
		t.Errorf("expected Accept '*/*', got '%s'", receivedAccept)
	}
}

func TestHttpFetcherClient_InvalidURL(t *testing.T) {
	client := NewHttpFetcherClient()
	ctx := context.Background()

	_, err := client.Fetch(ctx, "invalid-url")
	if err == nil {
		t.Errorf("expected error for invalid URL but got none")
	}
}

func TestHttpFetcherClient_RedirectNotFollowed(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/redirected")
		w.WriteHeader(http.StatusFound)
	}))
	defer redirectServer.Close()

	client := NewHttpFetcherClient()
	ctx := context.Background()

	response, err := client.Fetch(ctx, redirectServer.URL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, response.StatusCode)
	}
}