package main

import (
	"testing"

	"github.com/zerodayz1/silo-plugin-omdb/omdb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRatingsStruct_BothScores(t *testing.T) {
	r := &omdb.Ratings{IMDB: 8.7, RTCritic: 94}
	s := ratingsStruct(r)
	if s == nil {
		t.Fatal("expected non-nil struct")
	}
	fields := s.GetFields()
	if fields["imdb"].GetNumberValue() != 8.7 {
		t.Errorf("imdb: got %v, want 8.7", fields["imdb"])
	}
	if fields["rt_critic"].GetNumberValue() != 94 {
		t.Errorf("rt_critic: got %v, want 94", fields["rt_critic"])
	}
	if _, ok := fields["rt_audience"]; ok {
		t.Error("rt_audience should not be present")
	}
}

func TestRatingsStruct_Empty(t *testing.T) {
	if ratingsStruct(&omdb.Ratings{}) != nil {
		t.Error("expected nil struct for zero ratings")
	}
}

func TestStringMapFromStruct_Nil(t *testing.T) {
	m := stringMapFromStruct(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestStringMapFromStruct_Values(t *testing.T) {
	s, _ := structpb.NewStruct(map[string]any{"imdb": "tt1234567", "empty": ""})
	m := stringMapFromStruct(s)
	if m["imdb"] != "tt1234567" {
		t.Errorf("imdb: got %q, want tt1234567", m["imdb"])
	}
	if _, ok := m["empty"]; ok {
		t.Error("empty string should be omitted")
	}
}
