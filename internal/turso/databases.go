package turso

import (
	"fmt"
	"net/http"

	"github.com/chiselstrike/iku-turso-cli/internal/clients"
)

type Database struct {
	ID       string `json:"dbId"`
	Name     string
	Type     string
	Region   string
	Hostname string
}

type databases struct {
	c *clients.Client
}

var Databases = &databases{Client}

func (d *databases) List() ([]Database, error) {
	r, err := d.c.Get("/v2/databases", nil)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get database listing: %s", r.Status)
	}

	type ListResponse struct {
		Databases []Database `json:"databases"`
	}
	resp, err := Unmarshall[ListResponse](r)
	return resp.Databases, err
}

func (d *databases) Delete(database string) error {
	url := fmt.Sprintf("/v2/databases/%s", database)
	r, err := d.c.Delete(url, nil)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusNotFound {
		return fmt.Errorf("could not find database %s", database)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete database: %s", r.Status)
	}

	return nil
}
