package clients

import (
	"fmt"
	"net/http"
)

type Database struct {
	ID       string `json:"dbId"`
	Name     string
	Type     string
	Region   string
	Hostname string
}

type databases struct {
	c *client
}

var Databases = &databases{Turso}

func (d *databases) List() ([]Database, error) {
	r, err := d.c.Get("/v2/databases", nil)
	if err != nil {
		return nil, err
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get database listing: %s", r.Status)
	}
	defer r.Body.Close()

	type ListResponse struct {
		Databases []Database `json:"databases"`
	}
	resp, err := Unmarshall[ListResponse](r)
	return resp.Databases, err
}
