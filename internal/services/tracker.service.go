package services

import (
	"context"
	"fmt"
	"github.com/jorgejr568/wheregoes/internal/clients"
	"github.com/jorgejr568/wheregoes/internal/utils"
	urlPkg "net/url"
)

type TrackCheckpoint struct {
	Url    string `json:"url"`
	Status int    `json:"status"`
}

type TrackResponse struct {
	Url         string            `json:"url"`
	Checkpoints []TrackCheckpoint `json:"checkpoints"`
}

type TrackerService interface {
	Track(ctx context.Context, url string) (TrackResponse, error)
}

type defaultTrackerService struct {
	fetcher clients.FetcherClient
}

func (t *defaultTrackerService) Track(ctx context.Context, url string) (TrackResponse, error) {
	var checkpoints []TrackCheckpoint
	for {
		res, err := t.fetcher.Fetch(ctx, url)
		if err != nil {
			return TrackResponse{}, err
		}

		checkpoints = append(checkpoints, TrackCheckpoint{
			Url:    url,
			Status: res.StatusCode,
		})

		isRedirect := res.StatusCode >= 300 && res.StatusCode < 400
		if !isRedirect {
			break
		}

		url = t.transformLocationUrl(res.Headers.Get("Location"), url)
		if url == "" {
			break
		}

	}

	return TrackResponse{
		Url:         url,
		Checkpoints: checkpoints,
	}, nil
}

func (t *defaultTrackerService) transformLocationUrl(locationUrl string, previousUrl string) string {
	if utils.IsUrl(locationUrl) {
		return locationUrl
	}

	parsedPreviousUrl, err := urlPkg.Parse(previousUrl)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s://%s%s", parsedPreviousUrl.Scheme, parsedPreviousUrl.Host, locationUrl)
}

func NewTrackerService(fetcher clients.FetcherClient) TrackerService {
	return &defaultTrackerService{
		fetcher: fetcher,
	}
}
