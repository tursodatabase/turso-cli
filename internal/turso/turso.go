package turso

import (
	"fmt"
	"log"
	"net/url"

	"github.com/chiselstrike/iku-turso-cli/internal/clients"
)

var Client = initTurso()

func initTurso() *clients.Client {
	serialized := getTursoUrl()
	base, err := url.Parse(serialized)
	if err != nil {
		log.Fatalf("could not parse client api base url: %s", serialized)
	}
	token, err := getAccessToken()
	if err != nil {
		log.Fatalf(fmt.Errorf("could not parse access token: %w", err).Error())
	}
	return NewTurso(base, token)
}

func NewTurso(base *url.URL, token string) *clients.Client {
	return clients.New(base, token)
}
