package tvdb

import (
	"github.com/dghubble/sling"
	"net/http"
)

// Token struct
type Token struct {
	Token string `json:"token"`
}

// TokenService tv series service
type TokenService struct {
	sling *sling.Sling
	Auth  *Auth
	Token *Token
}

// newTokenService initialize a new TokenService
func newTokenService(sling *sling.Sling, auth *Auth) *TokenService {
	return &TokenService{
		sling: sling,
		Auth:  auth,
		Token: new(Token),
	}
}

// Login requests and applies a new Token to the base client
func (s *TokenService) Login() (*Token, error) {
	jsonError := &JSONError{}
	_, err := s.sling.New().Post("/login").BodyJSON(s.Auth).Receive(s.Token, jsonError)

	s.sling.Set("Authorization", "Bearer "+s.Token.Token)
	return s.Token, relevantError(err, jsonError)
}

// Refresh refreshes the stored token setting the new Authorization header
func (s *TokenService) Refresh() (*Token, *http.Response, error) {
	jsonError := &JSONError{}
	res, err := s.sling.New().Post("/refresh_token").Receive(s.Token, jsonError)

	s.sling.Set("Authorization", "Bearer "+s.Token.Token)
	return s.Token, res, relevantError(err, jsonError)
}
