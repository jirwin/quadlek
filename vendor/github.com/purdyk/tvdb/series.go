package tvdb

import (
	"fmt"
	"github.com/dghubble/sling"
)

// SeriesData response container
type SeriesData struct {
	Data   Series      `json:"data,omitempty"`
	Errors []JSONError `json:"errors,omitempty"`
}

// Series type
type Series struct {
	ID              int32    `json:"id"`
	SeriesName      string   `json:"seriesName"`
	Aliases         []string `json:"aliases,omitempty"`
	Banner          string   `json:"banner,omitempty"`
	SeriesID        string   `json:"seriesId,omitempty"`
	Status          string   `json:"status,omitempty"`
	FirstAired      string   `json:"firstAired,omitempty"`
	Network         string   `json:"network,omitempty"`
	NetworkID       string   `json:"networkId,omitempty"`
	Runtime         string   `json:"runtime,omitempty"`
	Genre           []string `json:"genre,omitempty"`
	Overview        string   `json:"overview,omitempty"`
	LastUpdated     int32    `json:"lastUpdated,omitempty"`
	AirsDayOfWeek   string   `json:"airsDayOfWeek,omitempty"`
	AirsTime        string   `json:"airsTime,omitempty"`
	Rating          string   `json:"rating,omitempty"`
	ImdbID          string   `json:"imdbId,omitempty"`
	Zap2itID        string   `json:"zap2itId,omitempty"`
	Added           string   `json:"added,omitempty"`
	SiteRating      float32  `json:"siteRating,omitempty"`
	SiteRatingCount int32    `json:"siteRatingCount,omitempty"`
}

// SeriesService TV Series service
type SeriesService struct {
	sling *sling.Sling
}

// NewSeriesService initialize a new SeriesService
func newSeriesService(sling *sling.Sling) *SeriesService {
	return &SeriesService{
		sling: sling,
	}
}

// Get one TV Serie by ID
func (s *SeriesService) Get(id int32) (*Series, error) {
	series := &SeriesData{}
	jsonError := &JSONError{}
	path := fmt.Sprintf("/series/%d", id)

	_, err := s.sling.New().Get(path).Receive(series, jsonError)
	return &series.Data, relevantError(err, jsonError)
}
