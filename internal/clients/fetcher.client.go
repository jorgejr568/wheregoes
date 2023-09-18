package clients

import (
	"context"
	"net/http"
)

type FetcherResponse struct {
	StatusCode int
	Headers    http.Header
}

type FetcherClient interface {
	Fetch(context context.Context, url string) (FetcherResponse, error)
}

type defaultHttpFetcherClient struct {
	client *http.Client
}

func (f *defaultHttpFetcherClient) Fetch(ctx context.Context, url string) (FetcherResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return FetcherResponse{}, err
	}

	req.Header.Add("User-Agent", "wheregoes")
	req.Header.Add("Accept", "*/*")
	res, err := f.client.Do(req)
	if err != nil {
		return FetcherResponse{}, err
	}

	return FetcherResponse{
		StatusCode: res.StatusCode,
		Headers:    res.Header,
	}, nil
}

func NewHttpFetcherClient() FetcherClient {
	return &defaultHttpFetcherClient{
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}
