package services

import (
	"context"
	"fmt"
	"github.com/jorgejr568/wheregoes/internal/clients"
	"github.com/jorgejr568/wheregoes/internal/utils"
	urlPkg "net/url"
	"time"
)

type TrackCheckpoint struct {
	Url     string        `json:"url"`
	Status  int           `json:"status"`
	Latency time.Duration `json:"latency"`
}

type TrackResponse struct {
	Url         string            `json:"url"`
	Checkpoints []TrackCheckpoint `json:"checkpoints"`
}

type TrackChannelResponse struct {
	Checkpoint *TrackCheckpoint
	Err        error
	Finished   bool
}

type TrackerService interface {
	Track(ctx context.Context, url string) (TrackResponse, error)
	TrackChannel(ctx context.Context, url string) <-chan TrackChannelResponse
}

type defaultTrackerService struct {
	fetcher clients.FetcherClient
}

func (t *defaultTrackerService) Track(ctx context.Context, url string) (TrackResponse, error) {
	var checkpoints []TrackCheckpoint
	for {
		now := time.Now()
		res, err := t.fetcher.Fetch(ctx, url)
		duration := time.Since(now)
		if err != nil {
			return TrackResponse{}, err
		}

		checkpoints = append(checkpoints, TrackCheckpoint{
			Url:     url,
			Latency: duration,
			Status:  res.StatusCode,
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

func (t *defaultTrackerService) TrackChannel(ctx context.Context, url string) <-chan TrackChannelResponse {
	ch := make(chan TrackChannelResponse)

	go func() {
		defer close(ch)

		for {
			now := time.Now()
			res, err := t.fetcher.Fetch(ctx, url)
			duration := time.Since(now)
			if err != nil {
				ch <- TrackChannelResponse{
					Err: err,
				}
				return
			}

			ch <- TrackChannelResponse{
				Checkpoint: &TrackCheckpoint{
					Url:     url,
					Latency: duration,
					Status:  res.StatusCode,
				},
			}

			isRedirect := res.StatusCode >= 300 && res.StatusCode < 400
			if !isRedirect {
				ch <- TrackChannelResponse{
					Finished: true,
				}
				return
			}

			url = t.transformLocationUrl(res.Headers.Get("Location"), url)
			if url == "" {
				ch <- TrackChannelResponse{
					Finished: true,
				}
				return
			}
		}
	}()

	return ch
}

func NewTrackerService(fetcher clients.FetcherClient) TrackerService {
	return &defaultTrackerService{
		fetcher: fetcher,
	}
}
