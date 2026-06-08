package omdb

// Response is the OMDb API JSON response for a single item lookup.
type Response struct {
	Response   string   `json:"Response"`   // "True" or "False"
	Error      string   `json:"Error"`      // populated when Response == "False"
	IMDbRating string   `json:"imdbRating"` // e.g. "8.7" or "N/A"
	Ratings    []Rating `json:"Ratings"`
}

// Rating is one entry in the OMDb Ratings array.
type Rating struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}

// Ratings holds the parsed numeric scores returned by a lookup.
type Ratings struct {
	IMDB     float64
	RTCritic int
}
