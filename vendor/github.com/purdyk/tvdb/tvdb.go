// Package tvdb a simple interface to thetvdb.com REST API
//
// Copyright 2016 Lorenzo Giuliani <lorenzo@frenzart.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tvdb

import (
	"github.com/dghubble/sling"
	"net/http"
)

const (
	baseURL = "https://api.thetvdb.com/"

	loginURL   = "/login"
	refreshURL = "/refresh_token"

	// Version of the client
	Version = "0.1.0"
)

// Client is the TvDB REST API client
type Client struct {
	sling *sling.Sling

	// Authentication configuration
	Auth *Auth

	// Services
	Search   *SearchService
	Series   *SeriesService
	Episodes *EpisodesService
	Token    *TokenService
}

// Auth authentication scheme
type Auth struct {
	APIKey   string
	UserKey  string
	Username string
}

// NewClient returns a new TvDB REST client
func NewClient(httpClient *http.Client, auth *Auth) *Client {
	base := sling.New()
	base.Client(httpClient).Base(baseURL).Set("user-agent", "tvdb.go client "+Version)

	return &Client{
		sling:    base,
		Auth:     auth,
		Search:   newSearchService(base),
		Series:   newSeriesService(base),
		Episodes: newEpisodesService(base),
		Token:    newTokenService(base, auth),
	}
}

// Login calls the Client.Token#Login method
func (c *Client) Login() error {
	_, err := c.Token.Login()
	return err
}
