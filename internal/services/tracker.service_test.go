package services

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/jorgejr568/wheregoes/internal/clients"
)

type mockFetcherClient struct {
	responses []clients.FetcherResponse
	errors    []error
	callCount int
}

func (m *mockFetcherClient) Fetch(ctx context.Context, url string) (clients.FetcherResponse, error) {
	if m.callCount >= len(m.responses) {
		if len(m.errors) > m.callCount {
			return clients.FetcherResponse{}, m.errors[m.callCount]
		}
		return clients.FetcherResponse{}, errors.New("unexpected call")
	}
	
	defer func() { m.callCount++ }()
	
	if len(m.errors) > m.callCount && m.errors[m.callCount] != nil {
		return clients.FetcherResponse{}, m.errors[m.callCount]
	}
	
	return m.responses[m.callCount], nil
}

func TestTrackerService_Track_NoRedirect(t *testing.T) {
	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{
			{
				StatusCode: http.StatusOK,
				Headers:    make(http.Header),
			},
		},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	response, err := service.Track(ctx, "https://example.com")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if response.Url != "https://example.com" {
		t.Errorf("expected final URL to be https://example.com, got %s", response.Url)
	}

	if len(response.Checkpoints) != 1 {
		t.Errorf("expected 1 checkpoint, got %d", len(response.Checkpoints))
	}

	if response.Checkpoints[0].Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", response.Checkpoints[0].Status)
	}
}

func TestTrackerService_Track_WithRedirect(t *testing.T) {
	headers1 := make(http.Header)
	headers1.Set("Location", "https://example.com/redirected")

	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{
			{
				StatusCode: http.StatusFound,
				Headers:    headers1,
			},
			{
				StatusCode: http.StatusOK,
				Headers:    make(http.Header),
			},
		},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	response, err := service.Track(ctx, "https://example.com")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if response.Url != "https://example.com/redirected" {
		t.Errorf("expected final URL to be https://example.com/redirected, got %s", response.Url)
	}

	if len(response.Checkpoints) != 2 {
		t.Errorf("expected 2 checkpoints, got %d", len(response.Checkpoints))
	}

	if response.Checkpoints[0].Status != http.StatusFound {
		t.Errorf("expected first checkpoint status 302, got %d", response.Checkpoints[0].Status)
	}

	if response.Checkpoints[1].Status != http.StatusOK {
		t.Errorf("expected second checkpoint status 200, got %d", response.Checkpoints[1].Status)
	}
}

func TestTrackerService_Track_RelativeRedirect(t *testing.T) {
	headers1 := make(http.Header)
	headers1.Set("Location", "/relative-path")

	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{
			{
				StatusCode: http.StatusFound,
				Headers:    headers1,
			},
			{
				StatusCode: http.StatusOK,
				Headers:    make(http.Header),
			},
		},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	response, err := service.Track(ctx, "https://example.com")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if response.Url != "https://example.com/relative-path" {
		t.Errorf("expected final URL to be https://example.com/relative-path, got %s", response.Url)
	}

	if len(response.Checkpoints) != 2 {
		t.Errorf("expected 2 checkpoints, got %d", len(response.Checkpoints))
	}
}

func TestTrackerService_Track_CircularRedirect(t *testing.T) {
	headers1 := make(http.Header)
	headers1.Set("Location", "https://example.com/b")
	headers2 := make(http.Header)
	headers2.Set("Location", "https://example.com/a")

	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{
			{
				StatusCode: http.StatusFound,
				Headers:    headers1,
			},
			{
				StatusCode: http.StatusFound,
				Headers:    headers2,
			},
			{
				StatusCode: http.StatusFound,
				Headers:    headers1,
			},
		},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	_, err := service.Track(ctx, "https://example.com/a")

	if err == nil {
		t.Errorf("expected circular redirect error")
	}

	if !errors.Is(err, ErrCircularRedirection) {
		t.Errorf("expected ErrCircularRedirection, got %v", err)
	}
}

func TestTrackerService_Track_FetchError(t *testing.T) {
	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{},
		errors:    []error{errors.New("network error")},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	_, err := service.Track(ctx, "https://example.com")

	if err == nil {
		t.Errorf("expected network error")
	}
}

func TestTrackerService_Track_EmptyLocationHeader(t *testing.T) {
	headers1 := make(http.Header)
	headers1.Set("Location", "")

	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{
			{
				StatusCode: http.StatusFound,
				Headers:    headers1,
			},
		},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	response, err := service.Track(ctx, "https://example.com")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(response.Checkpoints) == 0 {
		t.Errorf("expected at least 1 checkpoint, got %d", len(response.Checkpoints))
	}
}

func TestTrackerService_TrackChannel(t *testing.T) {
	headers1 := make(http.Header)
	headers1.Set("Location", "https://example.com/redirected")

	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{
			{
				StatusCode: http.StatusFound,
				Headers:    headers1,
			},
			{
				StatusCode: http.StatusOK,
				Headers:    make(http.Header),
			},
		},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	ch := service.TrackChannel(ctx, "https://example.com")

	var responses []TrackChannelResponse
	for response := range ch {
		responses = append(responses, response)
	}

	if len(responses) != 3 {
		t.Errorf("expected 3 responses (2 checkpoints + 1 finish), got %d", len(responses))
	}

	firstCheckpoint := responses[0]
	if firstCheckpoint.Checkpoint == nil {
		t.Errorf("expected first response to have checkpoint")
	}
	if firstCheckpoint.Checkpoint.Status != http.StatusFound {
		t.Errorf("expected first checkpoint status 302, got %d", firstCheckpoint.Checkpoint.Status)
	}

	secondCheckpoint := responses[1]
	if secondCheckpoint.Checkpoint == nil {
		t.Errorf("expected second response to have checkpoint")
	}
	if secondCheckpoint.Checkpoint.Status != http.StatusOK {
		t.Errorf("expected second checkpoint status 200, got %d", secondCheckpoint.Checkpoint.Status)
	}

	finishResponse := responses[2]
	if !finishResponse.Finished {
		t.Errorf("expected last response to be finished")
	}
}

func TestTrackerService_TrackChannel_Error(t *testing.T) {
	mockClient := &mockFetcherClient{
		responses: []clients.FetcherResponse{},
		errors:    []error{errors.New("network error")},
	}

	service := NewTrackerService(mockClient)
	ctx := context.Background()

	ch := service.TrackChannel(ctx, "https://example.com")

	var responses []TrackChannelResponse
	for response := range ch {
		responses = append(responses, response)
	}

	if len(responses) != 1 {
		t.Errorf("expected 1 error response, got %d", len(responses))
	}

	if responses[0].Err == nil {
		t.Errorf("expected error response")
	}
}

func TestTrackerService_transformLocationUrl(t *testing.T) {
	service := &defaultTrackerService{}

	tests := []struct {
		name         string
		locationUrl  string
		previousUrl  string
		expected     string
	}{
		{
			name:         "absolute URL",
			locationUrl:  "https://example.com/new",
			previousUrl:  "https://old.com",
			expected:     "https://example.com/new",
		},
		{
			name:         "relative path",
			locationUrl:  "/new-path",
			previousUrl:  "https://example.com/old",
			expected:     "https://example.com/new-path",
		},
		{
			name:         "relative path with query",
			locationUrl:  "/new-path?param=value",
			previousUrl:  "http://example.com:8080/old",
			expected:     "http://example.com:8080/new-path?param=value",
		},
		{
			name:         "invalid previous URL",
			locationUrl:  "/new-path",
			previousUrl:  "invalid-url",
			expected:     ":///new-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.transformLocationUrl(tt.locationUrl, tt.previousUrl)
			if result != tt.expected {
				t.Errorf("transformLocationUrl(%s, %s) = %s, expected %s", 
					tt.locationUrl, tt.previousUrl, result, tt.expected)
			}
		})
	}
}

func TestNewTrackerService(t *testing.T) {
	mockClient := &mockFetcherClient{}
	service := NewTrackerService(mockClient)

	if service == nil {
		t.Errorf("NewTrackerService returned nil")
	}

	if reflect.TypeOf(service).String() != "*services.defaultTrackerService" {
		t.Errorf("NewTrackerService returned wrong type")
	}
}