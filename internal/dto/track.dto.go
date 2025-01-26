package dto

import "github.com/jorgejr568/wheregoes/internal/services"

type trackResponseTypename string

const (
	trackResponseTypenameCheckpoint trackResponseTypename = "Checkpoint"
	trackResponseTypenameError      trackResponseTypename = "Error"
	trackResponseTypenameFinish     trackResponseTypename = "Finish"
)

type TrackRequest struct {
	Url string `json:"url"`
}

type trackResponse struct {
	TypeName trackResponseTypename `json:"__typename"`
}

type trackErrorResponse struct {
	trackResponse
	Error string `json:"error"`
}

type trackFinishResponse struct {
	trackResponse
}

type trackCheckpointResponse struct {
	trackResponse
	*services.TrackCheckpoint
}

func NewTrackErrorResponse(err error) trackErrorResponse {
	return trackErrorResponse{
		trackResponse: trackResponse{TypeName: trackResponseTypenameError},
		Error:         err.Error(),
	}
}

func NewTrackFinishResponse() trackFinishResponse {
	return trackFinishResponse{
		trackResponse: trackResponse{TypeName: trackResponseTypenameFinish},
	}
}

func NewTrackCheckpointResponse(checkpoint *services.TrackCheckpoint) trackCheckpointResponse {
	return trackCheckpointResponse{
		trackResponse:   trackResponse{TypeName: trackResponseTypenameCheckpoint},
		TrackCheckpoint: checkpoint,
	}
}
