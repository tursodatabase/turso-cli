package turso

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

type DatabasesClient client

func (d *DatabasesClient) List() ([]Database, error) {
	r, err := d.client.Get("/v2/databases", nil)
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

func (d *DatabasesClient) Delete(database string) error {
	url := fmt.Sprintf("/v2/databases/%s", database)
	r, err := d.client.Delete(url, nil)
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
