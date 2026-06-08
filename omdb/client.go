package omdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL  = "https://www.omdbapi.com/"
	maxResponseBody = 1 << 20 // 1 MB
)

// Client is an HTTP client for the OMDb API.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewClient returns a Client configured with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
	}
}

// SetBaseURL overrides the API base URL. Used for testing.
func (c *Client) SetBaseURL(u string) { c.baseURL = u }

// GetRatings fetches IMDb and Rotten Tomatoes critic scores for the given IMDb
// ID. Returns nil, nil when the title is not found or has no usable ratings —
// callers should treat nil as "no data" rather than an error.
func (c *Client) GetRatings(ctx context.Context, imdbID string) (*Ratings, error) {
	reqURL := c.baseURL + "?r=json&i=" + url.QueryEscape(imdbID) + "&apikey=" + url.QueryEscape(c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("omdb: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("omdb: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("omdb: invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("omdb: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("omdb: read response: %w", err)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("omdb: decode response: %w", err)
	}
	if result.Response != "True" {
		return nil, nil
	}

	ratings := &Ratings{}

	if result.IMDbRating != "" && result.IMDbRating != "N/A" {
		if v, err := strconv.ParseFloat(result.IMDbRating, 64); err == nil {
			ratings.IMDB = v
		}
	}

	for _, r := range result.Ratings {
		if r.Source == "Rotten Tomatoes" {
			trimmed := strings.TrimSuffix(r.Value, "%")
			if trimmed != r.Value {
				if v, err := strconv.Atoi(trimmed); err == nil {
					ratings.RTCritic = v
				}
			}
		}
	}

	if ratings.IMDB == 0 && ratings.RTCritic == 0 {
		return nil, nil
	}
	return ratings, nil
}
