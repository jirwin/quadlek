package tvdb

import (
	"github.com/dghubble/sling"
)

// SearchService provides an interface to the search engine
type SearchService struct {
	sling *sling.Sling
}

// NewSearchService returns a new SearchService
func newSearchService(sling *sling.Sling) *SearchService {
	return &SearchService{
		sling: sling,
	}
}

// SearchParams all optional search parameters
type SearchParams struct {
	Name     *string `url:"name,omitempty"`
	ImdbID   *string `url:"imdbId,omitempty"`
	Zap2itID *string `url:"zap2itId,omitempty"`
}

// SeriesSearchData type definition of search results
type SeriesSearchData struct {
	ID         int32    `json:"id,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Banner     string   `json:"banner,omitempty"`
	FirstAired string   `json:"firstAired,omitempty"`
	Network    string   `json:"network,omitempty"`
	Overview   string   `json:"overview,omitempty"`
	SeriesName string   `json:"seriesName,omitempty"`
	Status     string   `json:"status,omitempty"`
}

// SeriesSearchResults contains search results
type SeriesSearchResults struct {
	Data []*SeriesSearchData `json:"data,omitempty"`
}

// Search search by SearchParams
func (s *SearchService) Search(params *SearchParams) ([]*SeriesSearchData, error) {
	series := &SeriesSearchResults{}
	jsonError := &JSONError{}
	_, err := s.sling.New().Get("/search/series").QueryStruct(params).Receive(series, jsonError)
	return series.Data, relevantError(err, jsonError)
}

// ByName Search series by name
func (s *SearchService) ByName(name string) ([]*SeriesSearchData, error) {
	params := &SearchParams{}
	params.Name = &name
	return s.Search(params)
}

// ByImdbID Search by IMDB id
func (s *SearchService) ByImdbID(id string) ([]*SeriesSearchData, error) {
	params := &SearchParams{}
	params.ImdbID = &id
	return s.Search(params)
}

// ByZap2itID Search by Zap2it id
func (s *SearchService) ByZap2itID(id string) ([]*SeriesSearchData, error) {
	params := &SearchParams{}
	params.Zap2itID = &id
	return s.Search(params)
}
