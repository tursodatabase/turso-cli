package turso

import (
	"fmt"
	"net/http"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
)

type OrganizationsClient client

type Organization struct {
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
	Type string `json:"type,omitempty"`
}

func (c *OrganizationsClient) List() ([]Organization, error) {
	r, err := c.client.Get("/v2/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request organizations: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list organizations: %s", r.Status)

	}

	type ListResponse struct {
		Orgs []Organization `json:"organizations"`
	}

	data, err := unmarshal[ListResponse](r)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize list organizations response: %w", err)
	}

	return data.Orgs, nil
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

	if r.StatusCode == http.StatusConflict {
		return Organization{}, fmt.Errorf("failed to create organization %s: name already exists", internal.Emph(name))
	}

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

	switch r.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return parseResponseError(r)
	case http.StatusForbidden:
		return fmt.Errorf("you do not have permission to delete organization %s", slug)
	default:
		return fmt.Errorf("failed to delete organization: %s", r.Status)
	}
}

type OrgTotal struct {
	RowsRead         uint64 `json:"rows_read,omitempty"`
	RowsWritten      uint64 `json:"rows_written,omitempty"`
	StorageBytesUsed uint64 `json:"storage_bytes,omitempty"`
	Databases        uint64 `json:"databases,omitempty"`
	Locations        uint64 `json:"locations,omitempty"`
}

type OrgUsage struct {
	Databases map[string]DbUsage `json:"databases"`
	Total     OrgTotal           `json:"total"`
}

func (c *OrganizationsClient) Usage() (OrgUsage, error) {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}

	r, err := c.client.Get(prefix+"/usage", nil)
	if err != nil {
		return OrgUsage{}, fmt.Errorf("failed to get database usage: %w", err)
	}
	defer r.Body.Close()

	body, err := unmarshal[OrgUsage](r)
	return body, err
}

type Member struct {
	Name string `json:"username,omitempty"`
	Role string `json:"role,omitempty"`
}

func (c *OrganizationsClient) ListMembers() ([]Member, error) {
	url, err := c.MembersURL("")
	if err != nil {
		return nil, err
	}

	r, err := c.client.Get(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request organization members: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("only organization owners can list members")
	}

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
	url, err := c.MembersURL("")
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

	if r.StatusCode == http.StatusForbidden {
		return fmt.Errorf("only organization owners can add members")
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add organization member: %s", r.Status)
	}

	return nil
}

func (c *OrganizationsClient) RemoveMember(username string) error {
	url, err := c.MembersURL("/" + username)
	if err != nil {
		return err
	}

	r, err := c.client.Delete(url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization member: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusForbidden {
		return fmt.Errorf("only organization owners can remove members")
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove organization member: %s", r.Status)
	}

	return nil
}

func (c *OrganizationsClient) MembersURL(suffix string) (string, error) {
	if c.client.Org == "" {
		return "", fmt.Errorf("the currently active organization %s does not allow members. You can use %s to change active organization", internal.Emph("personal"), internal.Emph("turso org switch"))
	}
	return "/v1/organizations/" + c.client.Org + "/members" + suffix, nil
}

func unsetOrganization() error {
	settings, err := settings.ReadSettings()
	if err != nil {
		return err
	}
	settings.SetOrganization("")
	return nil
}

func isNotMemberErr(status int, org string) bool {
	if status == http.StatusForbidden && org != "" && unsetOrganization() == nil {
		return true
	}
	return false
}

func notMemberErr(org string) error {
	msg := fmt.Sprintf("you are not a member of organization %s. ", internal.Emph(org))
	msg += fmt.Sprintf("%s is now configured to use your personal organization.", internal.Emph("turso"))
	return fmt.Errorf(msg)
}
