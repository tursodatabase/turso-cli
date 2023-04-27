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

func (c *OrganizationsClient) Delete(slug string) error {
	r, err := c.client.Delete("/v1/organizations/"+slug, nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusNotFound {
		return fmt.Errorf("could not find organization %s", slug)
	}

	if r.StatusCode == http.StatusBadRequest {
		return fmt.Errorf("cannot delete personal organization %s", slug)
	}

	if r.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("you do not have permission to delete organization %s", slug)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete organization: %s", r.Status)
	}

	return nil
}

type Member struct {
	Name string `json:"username,omitempty"`
	Role string `json:"role,omitempty"`
}

func (c *OrganizationsClient) ListMembers() ([]Member, error) {
	url, err := c.MembersURL()
	if err != nil {
		return nil, err
	}

	r, err := c.client.Get(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request organization members: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list organization members: %s", r.Status)
	}

	data, err := unmarshal[struct{ Members []Member }](r)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize list organizations response: %w", err)
	}

	return data.Members, nil
}

func (c *OrganizationsClient) AddMember(username string) error {
	url, err := c.MembersURL()
	if err != nil {
		return err
	}

	body, err := marshal(Member{Name: username})
	if err != nil {
		return fmt.Errorf("failed to marshall add member request body: %s", err)
	}

	r, err := c.client.Post(url, body)
	if err != nil {
		return fmt.Errorf("failed to post organization member: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add organization member: %s", r.Status)
	}

	return nil
}

func (c *OrganizationsClient) MembersURL() (string, error) {
	if c.client.org == "" {
		return "", fmt.Errorf("cannot manage members of personal organization")
	}
	return "/v1/organizations/" + c.client.org + "/members", nil
}
