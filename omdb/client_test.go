package omdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, payload any) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("testkey")
	c.httpClient = srv.Client()
	c.SetBaseURL(srv.URL + "/")
	return c, srv
}

func TestGetRatings_Both(t *testing.T) {
	c, _ := serve(t, Response{
		Response:   "True",
		IMDbRating: "8.7",
		Ratings: []Rating{
			{Source: "Rotten Tomatoes", Value: "94%"},
		},
	})
	r, err := c.GetRatings(context.Background(), "tt1234567")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil ratings")
	}
	if r.IMDB != 8.7 {
		t.Errorf("IMDB: got %v, want 8.7", r.IMDB)
	}
	if r.RTCritic != 94 {
		t.Errorf("RTCritic: got %v, want 94", r.RTCritic)
	}
}

func TestGetRatings_NotFound(t *testing.T) {
	c, _ := serve(t, Response{Response: "False", Error: "Movie not found!"})
	r, err := c.GetRatings(context.Background(), "tt0000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != nil {
		t.Errorf("expected nil ratings, got %+v", r)
	}
}

func TestGetRatings_NAValues(t *testing.T) {
	c, _ := serve(t, Response{
		Response:   "True",
		IMDbRating: "N/A",
		Ratings:    []Rating{},
	})
	r, err := c.GetRatings(context.Background(), "tt1234567")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != nil {
		t.Errorf("expected nil for all-NA ratings, got %+v", r)
	}
}
