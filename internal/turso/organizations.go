package turso

import (
	"fmt"
	"net/http"
)

type OrganizationsClient client

type Organization struct {
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
	Type string `json:"type,omitempty"`
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

func (c *OrganizationsClient) Create(name string) (Organization, error) {
	body, err := marshal(Organization{Name: name})
	if err != nil {
		return Organization{}, fmt.Errorf("failed to marshall create org request body: %s", err)
	}

	r, err := c.client.Post("/v1/organizations", body)
	if err != nil {
		return Organization{}, fmt.Errorf("failed to post organization: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return Organization{}, fmt.Errorf("failed to create organization: %s", r.Status)
	}

	data, err := unmarshal[struct{ Org Organization }](r)
	if err != nil {
		return Organization{}, fmt.Errorf("failed to deserialize create organizations response: %w", err)
	}

	return data.Org, nil
}

func (c *OrganizationsClient) Delete(name string) error {
	r, err := c.client.Delete("/v1/organizations/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusNotFound {
		return fmt.Errorf("could not find organization %s", name)
	}

	if r.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("you do not have permission to delete organization %s", name)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete organization: %s", r.Status)
	}

	return nil
}
