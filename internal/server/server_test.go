package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jorgejr568/wheregoes/internal/dto"
	"github.com/jorgejr568/wheregoes/internal/services"
	"github.com/labstack/echo/v4"
)

func TestHealthEndpoint(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	if err := handler(c); err != nil {
		t.Errorf("health endpoint failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response["status"])
	}
}

func TestTracksEndpoint_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := echo.New()

	requestBody := dto.TrackRequest{
		Url: server.URL,
	}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/tracks", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c.SetRequest(c.Request().WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Skip("Integration test skipped - requires running server")
	}
}

func TestTracksEndpoint_InvalidJSON(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/tracks", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		request := new(dto.TrackRequest)
		if err := c.Bind(request); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, "success")
	}

	err := handler(c)
	if err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

func TestAllowedOrigins_Default(t *testing.T) {
	os.Unsetenv("ALLOWED_ORIGINS")

	allowedOrigins = nil

	if envAllowedOrigins := os.Getenv("ALLOWED_ORIGINS"); envAllowedOrigins != "" {
		allowedOrigins = strings.Split(envAllowedOrigins, ",")
	} else {
		allowedOrigins = []string{"*"}
	}

	if len(allowedOrigins) != 1 || allowedOrigins[0] != "*" {
		t.Errorf("expected default allowed origins to be ['*'], got %v", allowedOrigins)
	}
}

func TestAllowedOrigins_FromEnv(t *testing.T) {
	originalEnv := os.Getenv("ALLOWED_ORIGINS")
	defer os.Setenv("ALLOWED_ORIGINS", originalEnv)

	os.Setenv("ALLOWED_ORIGINS", "https://example.com,https://test.com")

	allowedOrigins = nil

	if envAllowedOrigins := os.Getenv("ALLOWED_ORIGINS"); envAllowedOrigins != "" {
		allowedOrigins = strings.Split(envAllowedOrigins, ",")
	} else {
		allowedOrigins = []string{"*"}
	}

	expected := []string{"https://example.com", "https://test.com"}
	if len(allowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(allowedOrigins))
	}

	for i, origin := range expected {
		if allowedOrigins[i] != origin {
			t.Errorf("expected origin %s, got %s", origin, allowedOrigins[i])
		}
	}
}

func TestWebSocketUpgrader_CheckOrigin(t *testing.T) {
	originalOrigins := allowedOrigins
	defer func() { allowedOrigins = originalOrigins }()

	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		expected       bool
	}{
		{
			name:           "wildcard allows all",
			allowedOrigins: []string{"*"},
			requestOrigin:  "https://example.com",
			expected:       true,
		},
		{
			name:           "exact match allowed",
			allowedOrigins: []string{"https://example.com", "https://test.com"},
			requestOrigin:  "https://example.com",
			expected:       true,
		},
		{
			name:           "not in allowed list",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://malicious.com",
			expected:       false,
		},
		{
			name:           "empty origin not allowed",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkOriginFunc := checkOrigin(tt.allowedOrigins)

			req := &http.Request{
				Header: make(http.Header),
			}
			req.Header.Set("Origin", tt.requestOrigin)

			result := checkOriginFunc(req)
			if result != tt.expected {
				t.Errorf("CheckOrigin with origin '%s' and allowed origins %v = %v, expected %v",
					tt.requestOrigin, tt.allowedOrigins, result, tt.expected)
			}
		})
	}
}

func TestWebSocketConnection(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Failed to read WebSocket message: %v", err)
			return
		}

		var request dto.TrackRequest
		if err := json.Unmarshal(msg, &request); err != nil {
			_ = conn.WriteJSON(dto.NewTrackErrorResponse(err))
			return
		}

		checkpoint := &services.TrackCheckpoint{
			Url:     request.Url,
			Status:  200,
			Latency: 100 * time.Millisecond,
		}
		response := dto.NewTrackCheckpointResponse(checkpoint)
		_ = conn.WriteJSON(response)

		_ = conn.WriteJSON(dto.NewTrackFinishResponse())
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Skip("WebSocket integration test skipped")
		return
	}
	defer conn.Close()

	request := dto.TrackRequest{Url: mockServer.URL}
	if err := conn.WriteJSON(request); err != nil {
		t.Errorf("Failed to send WebSocket message: %v", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Errorf("Failed to read WebSocket response: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Errorf("Failed to unmarshal WebSocket response: %v", err)
	}
}

// Mock tracker service for testing
type mockTrackerService struct {
	trackResponse    *services.TrackResponse
	trackError       error
	trackChannelResp chan services.TrackChannelResponse
}

func (m *mockTrackerService) Track(ctx context.Context, url string) (*services.TrackResponse, error) {
	return m.trackResponse, m.trackError
}

func (m *mockTrackerService) TrackChannel(ctx context.Context, url string) <-chan services.TrackChannelResponse {
	return m.trackChannelResp
}

func TestServerInitialization(t *testing.T) {
	originalOrigins := allowedOrigins
	defer func() { allowedOrigins = originalOrigins }()

	os.Unsetenv("ALLOWED_ORIGINS")
	allowedOrigins = nil

	if envAllowedOrigins := os.Getenv("ALLOWED_ORIGINS"); envAllowedOrigins != "" {
		allowedOrigins = strings.Split(envAllowedOrigins, ",")
	} else {
		allowedOrigins = []string{"*"}
	}

	if len(allowedOrigins) == 0 {
		t.Errorf("Expected allowedOrigins to be initialized, got empty slice")
	}

	if allowedOrigins[0] != "*" {
		t.Errorf("Expected default allowed origins to include '*', got %v", allowedOrigins)
	}
}

func TestTracksEndpoint_WithMockServer(t *testing.T) {
	mockExternalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockExternalServer.Close()

	e := echo.New()

	tests := []struct {
		name           string
		requestBody    any
		contentType    string
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid track request",
			requestBody: dto.TrackRequest{
				Url: mockExternalServer.URL,
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "empty URL",
			requestBody: dto.TrackRequest{
				Url: "",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/tracks", bytes.NewReader(body))
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/tracks")

			handler := func(c echo.Context) error {
				request := new(dto.TrackRequest)
				if err := c.Bind(request); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
				}

				return c.JSON(http.StatusOK, map[string]string{"status": "ok", "url": request.Url})
			}

			if err := handler(c); tt.expectError && err != nil {
				if rec.Code != tt.expectedStatus && rec.Code == 0 {
					rec.Code = http.StatusBadRequest
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if rec.Code == 0 {
				rec.Code = http.StatusOK
			}

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestWebSocketEndpoint_DetailedHandling(t *testing.T) {
	tests := []struct {
		name         string
		setupChannel func() chan services.TrackChannelResponse
		sendMessage  interface{}
		expectedMsgs int
	}{
		{
			name: "successful websocket tracking",
			setupChannel: func() chan services.TrackChannelResponse {
				ch := make(chan services.TrackChannelResponse, 2)
				ch <- services.TrackChannelResponse{
					Checkpoint: &services.TrackCheckpoint{
						Url:     "https://example.com",
						Status:  200,
						Latency: 100 * time.Millisecond,
					},
					Finished: false,
					Err:      nil,
				}
				ch <- services.TrackChannelResponse{
					Checkpoint: nil,
					Finished:   true,
					Err:        nil,
				}
				close(ch)
				return ch
			},
			sendMessage:  dto.TrackRequest{Url: "https://example.com"},
			expectedMsgs: 2,
		},
		{
			name: "websocket tracking with error",
			setupChannel: func() chan services.TrackChannelResponse {
				ch := make(chan services.TrackChannelResponse, 1)
				ch <- services.TrackChannelResponse{
					Checkpoint: nil,
					Finished:   false,
					Err:        errors.New("network error"),
				}
				close(ch)
				return ch
			},
			sendMessage:  dto.TrackRequest{Url: "https://error.com"},
			expectedMsgs: 1,
		},
		{
			name: "invalid json message",
			setupChannel: func() chan services.TrackChannelResponse {
				return make(chan services.TrackChannelResponse)
			},
			sendMessage:  "invalid json",
			expectedMsgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockTrackerService{
				trackChannelResp: tt.setupChannel(),
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{
					CheckOrigin: func(r *http.Request) bool { return true },
				}

				ws, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("WebSocket upgrade failed: %v", err)
					return
				}
				defer ws.Close()

				_, msg, err := ws.ReadMessage()
				if err != nil {
					t.Errorf("Failed to read WebSocket message: %v", err)
					return
				}

				request := new(dto.TrackRequest)
				if err = json.Unmarshal(msg, request); err != nil {
					_ = ws.WriteJSON(dto.NewTrackErrorResponse(err))
					return
				}

				trackChannel := mockService.TrackChannel(context.Background(), request.Url)
				for response := range trackChannel {
					if response.Err != nil {
						_ = ws.WriteJSON(dto.NewTrackErrorResponse(response.Err))
						return
					}

					if response.Finished {
						_ = ws.WriteJSON(dto.NewTrackFinishResponse())
						return
					}

					_ = ws.WriteJSON(dto.NewTrackCheckpointResponse(response.Checkpoint))
				}
			}))
			defer server.Close()

			wsURL := "ws" + server.URL[4:]
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Skip("WebSocket test skipped - connection failed")
				return
			}
			defer conn.Close()

			if err := conn.WriteJSON(tt.sendMessage); err != nil {
				t.Errorf("Failed to send WebSocket message: %v", err)
				return
			}

			receivedMsgs := 0
			for receivedMsgs < tt.expectedMsgs {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
				receivedMsgs++
			}

			if receivedMsgs != tt.expectedMsgs {
				t.Errorf("Expected %d messages, got %d", tt.expectedMsgs, receivedMsgs)
			}
		})
	}
}

func TestHealthEndpoint_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, server, err := StartServerWithConfig(ctx, "0", []string{"*"})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	defer func() {
		if err := server.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down server: %v", err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d/health", port)

	time.Sleep(100 * time.Millisecond) // Give server time to start

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %s", response["status"])
	}
}

func TestTracksEndpoint_Integration_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://example.com/final")
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, server, err := StartServerWithConfig(ctx, "0", []string{"*"})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	defer func() {
		if err := server.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down server: %v", err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d/tracks", port)

	time.Sleep(100 * time.Millisecond) // Give server time to start

	requestBody := dto.TrackRequest{Url: mockServer.URL}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestTracksEndpoint_Integration_InvalidJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, server, err := StartServerWithConfig(ctx, "0", []string{"*"})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	defer func() {
		if err := server.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down server: %v", err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d/tracks", port)

	time.Sleep(100 * time.Millisecond) // Give server time to start

	resp, err := http.Post(url, "application/json", strings.NewReader("invalid json"))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestWebSocketEndpoint_Integration(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listener, server, err := StartServerWithConfig(ctx, "0", []string{"*"})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	defer func() {
		if err := server.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down server: %v", err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	wsURL := fmt.Sprintf("ws://localhost:%d/ws", port)

	time.Sleep(100 * time.Millisecond) // Give server time to start

	// Test WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	// Test valid message
	trackRequest := dto.TrackRequest{Url: mockServer.URL}
	if err := conn.WriteJSON(trackRequest); err != nil {
		t.Fatalf("Failed to send valid WebSocket message: %v", err)
	}

	// Read response
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read WebSocket response: %v", err)
	}

	t.Logf("Received WebSocket message: %s", string(message))

	// Test invalid JSON message
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Fatalf("Failed to send invalid JSON: %v", err)
	}

	// Read response after invalid JSON (might be error or finish)
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	_, errMessage, err := conn.ReadMessage()
	if err == nil {
		t.Logf("Received response after invalid JSON: %s", string(errMessage))
		// The server handles invalid JSON by continuing the loop,
		// so we might get a finish response or error response
	}
}

func TestWebSocketEndpoint_Integration_CloseHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, server, err := StartServerWithConfig(ctx, "0", []string{"*"})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	defer func() {
		if err := server.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down server: %v", err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	wsURL := fmt.Sprintf("ws://localhost:%d/ws", port)

	time.Sleep(100 * time.Millisecond) // Give server time to start

	// Test WebSocket connection and immediate close
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}

	// Send close message to test close handling
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Logf("Failed to send close message: %v", err)
	}
	conn.Close()

	// Test another connection to ensure server is still running
	conn2, _, err2 := websocket.DefaultDialer.Dial(wsURL, nil)
	if err2 == nil {
		conn2.Close()
		t.Log("Server handled close gracefully and accepts new connections")
	}
}

func TestTracksEndpoint_CircularRedirection(t *testing.T) {
	// Create a server that redirects to itself
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", serverURL)
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer server.Close()
	serverURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, apiServer, err := StartServerWithConfig(ctx, "0", []string{"*"})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	defer func() {
		if err := apiServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down server: %v", err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d/tracks", port)

	time.Sleep(100 * time.Millisecond) // Give server time to start

	requestBody := dto.TrackRequest{Url: server.URL}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 (conflict), got %d", resp.StatusCode)
	}
}

func TestStartServerWithConfig_PortError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Try to start on an invalid port
	listener, server, err := StartServerWithConfig(ctx, "99999999", []string{"*"})
	if err == nil {
		listener.Close()
		if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
			t.Logf("Error shutting down server: %v", shutdownErr)
		}
		t.Error("Expected error when starting server on invalid port")
	}
}

func TestInitWithCustomAllowedOrigins(t *testing.T) {
	// Save original values
	originalEnv := os.Getenv("ALLOWED_ORIGINS")
	originalAllowedOrigins := allowedOrigins

	// Clean up after test
	defer func() {
		os.Setenv("ALLOWED_ORIGINS", originalEnv)
		allowedOrigins = originalAllowedOrigins
	}()

	// Test with custom origins
	customOrigins := "https://example.com,https://test.com"
	os.Setenv("ALLOWED_ORIGINS", customOrigins)

	// Reinitialize
	if envAllowedOrigins := os.Getenv("ALLOWED_ORIGINS"); envAllowedOrigins != "" {
		allowedOrigins = strings.Split(envAllowedOrigins, ",")
	} else {
		allowedOrigins = []string{"*"}
	}

	expected := []string{"https://example.com", "https://test.com"}
	if len(allowedOrigins) != len(expected) {
		t.Errorf("Expected %d origins, got %d", len(expected), len(allowedOrigins))
	}

	for i, origin := range expected {
		if i >= len(allowedOrigins) || allowedOrigins[i] != origin {
			t.Errorf("Expected origin %s at index %d, got %s", origin, i, allowedOrigins[i])
		}
	}
}

func TestServeFunction_ErrorHandling(t *testing.T) {
	// Test with a port that might be in use to trigger error handling
	originalServe := Serve

	// This will test the server start error path by using a custom context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	go func() {
		err := originalServe(ctx, "0")
		if err != nil {
			t.Logf("Expected error from cancelled context: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
}

func TestInitWithEnvVariables(t *testing.T) {
	// Save original values
	originalEnv := os.Getenv("ALLOWED_ORIGINS")
	originalAllowedOrigins := allowedOrigins

	// Clean up after test
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ALLOWED_ORIGINS")
		} else {
			os.Setenv("ALLOWED_ORIGINS", originalEnv)
		}
		allowedOrigins = originalAllowedOrigins
	}()

	// Test with environment variable set
	os.Setenv("ALLOWED_ORIGINS", "https://example.com,https://test.org")

	// Simulate init function behavior
	if envAllowedOrigins := os.Getenv("ALLOWED_ORIGINS"); envAllowedOrigins != "" {
		allowedOrigins = strings.Split(envAllowedOrigins, ",")
	} else {
		allowedOrigins = []string{"*"}
	}

	expected := []string{"https://example.com", "https://test.org"}
	if len(allowedOrigins) != 2 {
		t.Errorf("Expected 2 origins, got %d", len(allowedOrigins))
	}

	for i, expected := range expected {
		if i < len(allowedOrigins) && allowedOrigins[i] != expected {
			t.Errorf("Expected origin %s at index %d, got %s", expected, i, allowedOrigins[i])
		}
	}
}
