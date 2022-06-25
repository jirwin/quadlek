package gifs

import (
	"strings"

	"github.com/peterhellberg/giphy"
)

type Gifs struct {
	client *giphy.Client
}

func (g *Gifs) Search(query string) (string, error) {
	r, err := g.client.Search(strings.Split(query, " "))
	if err != nil {
		return "", err
	}

	return r.Data[0].MediaURL(), nil
}

func (g *Gifs) Translate(query string) (string, error) {
	r, err := g.client.Translate(strings.Split(query, " "))
	if err != nil {
		return "", err
	}

	return r.MediaURL(), nil
}

func (g *Gifs) Random(query string) (string, error) {
	r, err := g.client.Random(strings.Split(query, " "))
	if err != nil {
		return "", err
	}

	return r.Data.MediaURL(), nil
}

func NewGifs(apiKey, rating string) *Gifs {
	client := giphy.NewClient()
	client.APIKey = apiKey
	client.Rating = rating

	return &Gifs{
		client: client,
	}
}
