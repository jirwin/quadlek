package tvdb

import (
	"fmt"
	"time"

	"github.com/dghubble/sling"
)

// EpisodeRecordData Searches results
type EpisodeRecordData struct {
	Data   Episode   `json:"data,omitempty"`
	Errors JSONError `json:"jsonError,omitempty"`
}

// Episode a single episode
type Episode struct {
	ID                 int32    `json:"id,omitempty"`
	AiredSeason        int32    `json:"airedSeason,omitempty"`
	AiredEpisodeNumber int32    `json:"airedEpisodeNumber,omitempty"`
	EpisodeName        string   `json:"episodeName,omitempty"`
	FirstAired         string   `json:"firstAired,omitempty"`
	GuestStars         []string `json:"guestStars,omitempty"`
	Director           string   `json:"director,omitempty"`
	Directors          []string `json:"directors,omitempty"`
	Writers            []string `json:"writers,omitempty"`
	Overview           string   `json:"overview,omitempty"`
	ProductionCode     string   `json:"productionCode,omitempty"`
	ShowURL            string   `json:"showUrl,omitempty"`
	LastUpdated        int32    `json:"lastUpdated,omitempty"`
	DvdDiscid          string   `json:"dvdDiscid,omitempty"`
	DvdSeason          int32    `json:"dvdSeason,omitempty"`
	DvdEpisodeNumber   float32  `json:"dvdEpisodeNumber,omitempty"`
	DvdChapter         float32  `json:"dvdChapter,omitempty"`
	AbsoluteNumber     int32    `json:"absoluteNumber,omitempty"`
	Filename           string   `json:"filename,omitempty"`
	SeriesID           string   `json:"seriesId,omitempty"`
	LastUpdatedBy      string   `json:"lastUpdatedBy,omitempty"`
	AirsAfterSeason    int32    `json:"airsAfterSeason,omitempty"`
	AirsBeforeSeason   int32    `json:"airsBeforeSeason,omitempty"`
	AirsBeforeEpisode  int32    `json:"airsBeforeEpisode,omitempty"`
	ThumbAuthor        int32    `json:"thumbAuthor,omitempty"`
	ThumbAdded         string   `json:"thumbAdded,omitempty"`
	ThumbWidth         string   `json:"thumbWidth,omitempty"`
	ThumbHeight        string   `json:"thumbHeight,omitempty"`
	ImdbID             string   `json:"imdbId,omitempty"`
	SiteRating         float32  `json:"siteRating,omitempty"`
	SiteRatingCount    int32    `json:"siteRatingCount,omitempty"`
}

// SearchParams optional episodes search parameters
type EpisodeSearchParams struct {
	AbsoluteNumber *string `url:"absoluteNumber,omitempty"`
	AiredSeason    *string `url:"airedSeason,omitempty"`
	AiredEpisode   *string `url:"airedEpisode,omitempty"`
	DvdSeason      *string `url:"dvdSeason,omitempty"`
	DvdEpisode     *string `url:"dvdEpisode,omitempty"`
	ImdbId         *string `url:"imdbId,omitempty"`
	Page           *string `url:"page,omitempty"`
}

// Episodes Search Results
type EpisodesRecordData struct {
	Data  []Episode `json:"data,omitempty"`
	Links Links  `json:"links,omitempty"`
}

// EpisodesService the episode service
type EpisodesService struct {
	sling *sling.Sling
}

// newSeriesService initialize a new SeriesService
func newEpisodesService(sling *sling.Sling) *EpisodesService {
	return &EpisodesService{
		sling: sling,
	}
}

// Get a single episode
func (s *EpisodesService) Get(id int32) (*Episode, error) {
	episode := &Episode{}
	jsonError := &JSONError{}

	path := fmt.Sprintf("/episodes/%d", id)
	_, err := s.sling.New().Path(path).Receive(episode, jsonError)
	return episode, relevantError(err, jsonError)
}

// Note, only use the page value in EpisodeSearchParams
func (s *EpisodesService) ListEpisodes(seriesId int32, params *EpisodeSearchParams) (*EpisodesRecordData, error) {
	data := &EpisodesRecordData{}
	jsonError := &JSONError{}

	path := fmt.Sprintf("/series/%d/episodes", seriesId)
	_, e := s.sling.New().Get(path).QueryStruct(params).Receive(data, jsonError)

	return data, relevantError(e, jsonError)
}

// Find episodes meeting certain criteria
func (s *EpisodesService) SearchEpisodes(seriesId int32, params *EpisodeSearchParams) (*EpisodesRecordData, error) {
	data := &EpisodesRecordData{}
	jsonError := &JSONError{}

	path := fmt.Sprintf("/series/%d/episodes/query", seriesId)
	_, e := s.sling.New().Get(path).QueryStruct(params).Receive(data, jsonError)
	return data, relevantError(e, jsonError)
}

// Check if an episode is in the future
func (e *Episode) IsInFuture() (bool) {
	aired := e.ParseAired()
	if aired == nil {
		return true
	}

	now := time.Now()
	return aired.After(now)
}

// I guess date/time parsing is difficult in go
func (e *Episode) ParseAired() (*time.Time) {
	if e.FirstAired == "" {
		return nil
	}

	here, _ := time.LoadLocation("Local")
	var mm time.Month
	var dd int
	var yy int

	fmt.Sscanf(e.FirstAired, "%d-%d-%d", &yy, &mm, &dd)

	tt := time.Date(yy, mm, dd, 0, 0, 0, 0, here)
	return &tt
}
