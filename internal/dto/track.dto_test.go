package dto

import (
	"errors"
	"testing"

	"github.com/jorgejr568/wheregoes/internal/services"
)

func TestNewTrackErrorResponse(t *testing.T) {
	err := errors.New("test error")
	response := NewTrackErrorResponse(err)
	
	if response.Error != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", response.Error)
	}
	
	if response.TypeName != trackResponseTypenameError {
		t.Errorf("Expected typename %s, got %s", trackResponseTypenameError, response.TypeName)
	}
}

func TestNewTrackFinishResponse(t *testing.T) {
	response := NewTrackFinishResponse()
	
	if response.TypeName != trackResponseTypenameFinish {
		t.Errorf("Expected typename %s, got %s", trackResponseTypenameFinish, response.TypeName)
	}
}

func TestNewTrackCheckpointResponse(t *testing.T) {
	checkpoint := &services.TrackCheckpoint{
		Url:     "https://example.com",
		Status:  200,
		Latency: 100,
	}
	
	response := NewTrackCheckpointResponse(checkpoint)
	
	if response.TypeName != trackResponseTypenameCheckpoint {
		t.Errorf("Expected typename %s, got %s", trackResponseTypenameCheckpoint, response.TypeName)
	}
	
	if response.TrackCheckpoint.Url != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", response.TrackCheckpoint.Url)
	}
	
	if response.TrackCheckpoint.Status != 200 {
		t.Errorf("Expected status 200, got %d", response.TrackCheckpoint.Status)
	}
}