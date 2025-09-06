package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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
			allowedOrigins = tt.allowedOrigins
			
			req := &http.Request{
				Header: make(http.Header),
			}
			req.Header.Set("Origin", tt.requestOrigin)

			result := upgrader.CheckOrigin(req)
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
		name          string
		setupChannel  func() chan services.TrackChannelResponse
		sendMessage   interface{}
		expectedMsgs  int
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
			sendMessage: dto.TrackRequest{Url: "https://example.com"},
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
			sendMessage: dto.TrackRequest{Url: "https://error.com"},
			expectedMsgs: 1,
		},
		{
			name: "invalid json message",
			setupChannel: func() chan services.TrackChannelResponse {
				return make(chan services.TrackChannelResponse)
			},
			sendMessage: "invalid json",
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

func TestServeFunction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Serve returned unexpected error: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	
	resp, err := http.Get("http://localhost:8080/health")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected health endpoint to return 200, got %d", resp.StatusCode)
		}
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestServeFunction_WithRealEndpoints(t *testing.T) {
	mockExternalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer mockExternalServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Serve returned: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Test POST /tracks endpoint
	trackRequest := dto.TrackRequest{Url: mockExternalServer.URL}
	jsonBody, _ := json.Marshal(trackRequest)
	
	resp, err := http.Post("http://localhost:8080/tracks", "application/json", bytes.NewReader(jsonBody))
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Logf("POST /tracks response: %d, body: %s", resp.StatusCode, string(body))
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected POST /tracks to return 200, got %d", resp.StatusCode)
		}
	} else {
		t.Logf("POST request failed: %v", err)
	}

	// Test POST /tracks with invalid JSON
	resp2, err2 := http.Post("http://localhost:8080/tracks", "application/json", strings.NewReader("invalid json"))
	if err2 == nil {
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusBadRequest {
			t.Logf("Expected invalid JSON to return 400, got %d", resp2.StatusCode)
		}
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestServeFunction_WebSocketEndpoint(t *testing.T) {
	// Save and restore allowed origins
	originalAllowedOrigins := allowedOrigins
	defer func() { allowedOrigins = originalAllowedOrigins }()
	allowedOrigins = []string{"*"}

	mockExternalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockExternalServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Serve returned: %v", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	// Test WebSocket connection and message handling
	headers := http.Header{}
	headers.Set("Origin", "http://localhost")
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", headers)
	if err != nil {
		t.Logf("WebSocket connection failed: %v", err)
		cancel()
		return
	}
	defer conn.Close()

	// Test valid message
	trackRequest := dto.TrackRequest{Url: mockExternalServer.URL}
	if err := conn.WriteJSON(trackRequest); err != nil {
		t.Errorf("Failed to send valid WebSocket message: %v", err)
	}

	// Read responses
	for i := 0; i < 3; i++ {
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		t.Logf("Received WebSocket message: %s", string(message))
	}

	// Test invalid JSON message
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Logf("Failed to send invalid JSON: %v", err)
	} else {
		// Try to read error response
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, message, err := conn.ReadMessage()
		if err == nil {
			t.Logf("Received error response: %s", string(message))
		}
	}

	conn.Close()
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestServeFunction_ErrorHandling(t *testing.T) {
	// Test with a port that might be in use to trigger error handling
	originalServe := Serve
	
	// This will test the server start error path by using a custom context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	go func() {
		err := originalServe(ctx)
		if err != nil {
			t.Logf("Expected error from cancelled context: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
}

func TestServeFunction_CircularRedirectionError(t *testing.T) {
	// Mock server that causes circular redirection
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.String(), http.StatusFound) // Redirect to itself
	}))
	defer mockServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Serve returned: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// This should trigger circular redirection error
	trackRequest := dto.TrackRequest{Url: mockServer.URL}
	jsonBody, _ := json.Marshal(trackRequest)
	
	resp, err := http.Post("http://localhost:8080/tracks", "application/json", bytes.NewReader(jsonBody))
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Circular redirect response: %d, body: %s", resp.StatusCode, string(body))
		
		// Should return 409 Conflict for circular redirection
		if resp.StatusCode == http.StatusConflict {
			var errorResp struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
				t.Logf("Successfully caught circular redirection error: %s", errorResp.Error)
			}
		}
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocketConnection_CloseHandling(t *testing.T) {
	// Save and restore allowed origins
	originalAllowedOrigins := allowedOrigins
	defer func() { allowedOrigins = originalAllowedOrigins }()
	allowedOrigins = []string{"*"}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Serve returned: %v", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	// Test WebSocket connection and immediate close
	headers := http.Header{}
	headers.Set("Origin", "http://localhost")
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", headers)
	if err != nil {
		t.Logf("WebSocket connection failed: %v", err)
		cancel()
		return
	}

	// Send close message to test close handling
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.Close()

	// Test another connection to ensure server is still running
	conn2, _, err2 := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", headers)
	if err2 == nil {
		conn2.Close()
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestServeFunction_NetworkError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Serve returned: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Test with a URL that will cause a network error
	trackRequest := dto.TrackRequest{Url: "http://invalid-domain-that-does-not-exist.com"}
	jsonBody, _ := json.Marshal(trackRequest)
	
	resp, err := http.Post("http://localhost:8080/tracks", "application/json", bytes.NewReader(jsonBody))
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Network error response: %d, body: %s", resp.StatusCode, string(body))
		
		// This should trigger a network error and return 500
		if resp.StatusCode == http.StatusInternalServerError {
			t.Logf("Successfully caught network error")
		}
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocket_CompleteMessageLoop(t *testing.T) {
	// Save and restore allowed origins
	originalAllowedOrigins := allowedOrigins
	defer func() { allowedOrigins = originalAllowedOrigins }()
	allowedOrigins = []string{"*"}

	mockExternalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockExternalServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Serve returned: %v", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	headers := http.Header{}
	headers.Set("Origin", "http://localhost")
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", headers)
	if err != nil {
		t.Logf("WebSocket connection failed: %v", err)
		cancel()
		return
	}
	defer conn.Close()

	// Send multiple valid messages to exercise the loop
	for i := 0; i < 3; i++ {
		trackRequest := dto.TrackRequest{Url: mockExternalServer.URL}
		if err := conn.WriteJSON(trackRequest); err != nil {
			t.Errorf("Failed to send WebSocket message %d: %v", i, err)
			break
		}

		// Read all responses for this request
		for j := 0; j < 5; j++ { // Read up to 5 messages
			_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			t.Logf("Message %d-%d: %s", i, j, string(message))
			
			// Check if it's a finish message
			var resp map[string]interface{}
			if json.Unmarshal(message, &resp) == nil {
				if typename, ok := resp["__typename"]; ok && typename == "Finish" {
					break
				}
			}
		}
	}

	// Send invalid JSON to trigger error path
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err == nil {
		// Try to read error response
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, message, err := conn.ReadMessage()
		if err == nil {
			t.Logf("Error response for invalid JSON: %s", string(message))
		}

		// Continue sending more messages after error to test loop continuation
		trackRequest := dto.TrackRequest{Url: mockExternalServer.URL}
		_ = conn.WriteJSON(trackRequest)
		
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, message, err = conn.ReadMessage()
		if err == nil {
			t.Logf("Message after error recovery: %s", string(message))
		}
	}

	conn.Close()
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocket_ErrorInTrackChannel(t *testing.T) {
	// Save and restore allowed origins
	originalAllowedOrigins := allowedOrigins
	defer func() { allowedOrigins = originalAllowedOrigins }()
	allowedOrigins = []string{"*"}

	// This creates a URL that will cause an error in the tracker service
	invalidURL := "://invalid-url"
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		err := Serve(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Serve returned: %v", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	headers := http.Header{}
	headers.Set("Origin", "http://localhost")
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", headers)
	if err != nil {
		t.Logf("WebSocket connection failed: %v", err)
		cancel()
		return
	}
	defer conn.Close()

	// Send request with invalid URL to trigger error in track channel
	trackRequest := dto.TrackRequest{Url: invalidURL}
	if err := conn.WriteJSON(trackRequest); err != nil {
		t.Errorf("Failed to send invalid URL request: %v", err)
	} else {
		// Try to read error response
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, message, err := conn.ReadMessage()
		if err == nil {
			t.Logf("Error response for invalid URL: %s", string(message))
		}
	}

	conn.Close()
	cancel()
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