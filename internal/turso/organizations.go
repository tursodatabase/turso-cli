package turso

import (
	"fmt"
	"net/http"
)

type OrganizationsClient client

type Organization struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

func (c *OrganizationsClient) List() ([]Organization, error) {
	r, err := c.client.Get("/v1/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request organizations: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list organizations: %s", r.Status)

	}

	data, err := unmarshal[[]Organization](r)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize list organizations response: %w", err)
	}

	return data, nil
}
