package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
)

func getTursoUrl() string {
	host := os.Getenv("TURSO_API_BASEURL")
	if host == "" {
		host = "https://api.chiseledge.com"
	}
	return host
}

func Unmarshall[T any](r *http.Response) (T, error) {
	d, err := ioutil.ReadAll(r.Body)
	t := new(T)
	if err != nil {
		return *t, err
	}
	json.Unmarshal(d, &t)
	return *t, nil
}

func getAccessToken() (string, error) {
	settings, err := settings.ReadSettings()
	if err != nil {
		return "", fmt.Errorf("could not read local settings")
	}

	token := settings.GetToken()
	if token == "" {
		return "", fmt.Errorf("user not logged in")
	}

	return token, nil
}
