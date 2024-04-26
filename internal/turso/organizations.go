package turso

import (
	"fmt"
	"net/http"

	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/settings"
)

type OrganizationsClient client

type Organization struct {
	Name     string `json:"name,omitempty"`
	Slug     string `json:"slug,omitempty"`
	Type     string `json:"type,omitempty"`
	StripeID string `json:"stripe_id,omitempty"`
	Overages bool   `json:"overages,omitempty"`
}

func (c *OrganizationsClient) List() ([]Organization, error) {
	r, err := c.client.Get("/v2/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request organizations: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list organizations: %w", parseResponseError(r))
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

func (c *OrganizationsClient) Create(name string, stripeId string, dryRun bool) (Organization, error) {
	body, err := marshal(Organization{Name: name, StripeID: stripeId})
	if err != nil {
		return Organization{}, fmt.Errorf("failed to marshall create org request body: %s", err)
	}

	r, err := c.client.Post(fmt.Sprintf("/v1/organizations?dry_run=%v", dryRun), body)
	if err != nil {
		return Organization{}, fmt.Errorf("failed to post organization: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusConflict {
		return Organization{}, fmt.Errorf("failed to create organization %s: name already exists", internal.Emph(name))
	}

	if r.StatusCode == http.StatusPaymentRequired {
		return Organization{}, fmt.Errorf("failed to create organization %s: you need to upgrade your plan", internal.Emph(name))
	}

	if r.StatusCode != http.StatusOK {
		return Organization{}, fmt.Errorf("failed to create organization: %w", parseResponseError(r))
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
		return fmt.Errorf("failed to delete organization: %w", parseResponseError(r))
	}
}

type OrgTotal struct {
	RowsRead         uint64 `json:"rows_read,omitempty"`
	RowsWritten      uint64 `json:"rows_written,omitempty"`
	StorageBytesUsed uint64 `json:"storage_bytes,omitempty"`
	BytesSynced      uint64 `json:"bytes_synced,omitempty"`
	Databases        uint64 `json:"databases,omitempty"`
	Locations        uint64 `json:"locations,omitempty"`
	Groups           uint64 `json:"groups,omitempty"`
}

type OrgUsage struct {
	UUID      string    `json:"uuid,omitempty"`
	Usage     OrgTotal  `json:"usage"`
	Databases []DbUsage `json:"databases"`
}

type OrgUsageResponse struct {
	OrgUsage OrgUsage `json:"organization"`
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

	if r.StatusCode != http.StatusOK {
		err, _ := unmarshal[string](r)
		return OrgUsage{}, fmt.Errorf("failed to get database usage: %d %s", r.StatusCode, err)
	}

	body, err := unmarshal[OrgUsageResponse](r)
	if err != nil {
		return OrgUsage{}, err
	}
	return body.OrgUsage, nil
}

func (c *OrganizationsClient) SetOverages(slug string, toggle bool) error {
	path := "/v1/organizations/" + slug
	body, err := marshal(map[string]bool{"overages": toggle})
	if err != nil {
		return fmt.Errorf("failed to marshall set overages request body: %s", err)
	}
	r, err := c.client.Patch(path, body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set overages: %w", parseResponseError(r))
	}

	return nil
}

type Member struct {
	Name string `json:"username,omitempty"`
	Role string `json:"role,omitempty"`
}

type Invite struct {
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
	Accepted bool   `json:"accepted,omitempty"`
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
		return nil, fmt.Errorf("only organization admins or owners can list members")
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list organization members: %w", parseResponseError(r))
	}

	data, err := unmarshal[struct{ Members []Member }](r)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize list organizations response: %w", err)
	}

	return data.Members, nil
}

func (c *OrganizationsClient) AddMember(username, role string) error {
	url, err := c.MembersURL("")
	if err != nil {
		return err
	}

	body, err := marshal(Member{Name: username, Role: role})
	if err != nil {
		return fmt.Errorf("failed to marshall add member request body: %s", err)
	}

	r, err := c.client.Post(url, body)
	if err != nil {
		return fmt.Errorf("failed to post organization member: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusForbidden {
		return fmt.Errorf("only organization admins or owners can add members")
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add organization member: %w", parseResponseError(r))
	}

	return nil
}

func (c *OrganizationsClient) InviteMember(email, role string) error {
	prefix := "/v1/organizations/" + c.client.Org

	body, err := marshal(Invite{Email: email, Role: role})
	if err != nil {
		return fmt.Errorf("failed to marshall invite email request body: %s", err)
	}

	r, err := c.client.Post(prefix+"/invite", body)
	if err != nil {
		return fmt.Errorf("failed to invite organization member: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusForbidden {
		return fmt.Errorf("only organization admins or owners can invite members")
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to invite organization member: %w", parseResponseError(r))
	}

	return nil
}

func (c *OrganizationsClient) DeleteInvite(email string) error {
	prefix := "/v1/organizations/" + c.client.Org

	r, err := c.client.Delete(prefix+"/invites/"+email, nil)
	if err != nil {
		return fmt.Errorf("failed to remove pending invite: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusForbidden {
		return fmt.Errorf("only organization admins or owners can invite members")
	}

	if r.StatusCode == http.StatusNotFound {
		return fmt.Errorf("invite for %s not found", email)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete pending invite: %w", parseResponseError(r))
	}

	return nil
}

func (c *OrganizationsClient) ListInvites() ([]Invite, error) {
	prefix := "/v1/organizations/" + c.client.Org

	r, err := c.client.Get(prefix+"/invites", nil)
	if err != nil {
		return []Invite{}, fmt.Errorf("failed to list invites: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusForbidden {
		return []Invite{}, fmt.Errorf("only organization admins or owners can list invites")
	}

	if r.StatusCode != http.StatusOK {
		return []Invite{}, fmt.Errorf("failed to list invites: %w", parseResponseError(r))
	}

	data, err := unmarshal[struct {
		Invites []Invite `json:"invites"`
	}](r)
	if err != nil {
		return []Invite{}, fmt.Errorf("failed to deserialize list invites response: %w", err)
	}

	return data.Invites, nil
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
		return fmt.Errorf("only organization admins or owners can remove members")
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove organization member: %w", parseResponseError(r))
	}

	return nil
}

func (c *OrganizationsClient) MembersURL(suffix string) (string, error) {
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
