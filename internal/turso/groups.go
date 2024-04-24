package turso

import (
	"fmt"
	"net/http"

	"github.com/tursodatabase/turso-cli/internal"
)

type GroupsClient client

type Group struct {
	Name      string   `json:"name"`
	Locations []string `json:"locations"`
	Primary   string   `json:"primary"`
	Archived  bool     `json:"archived"`
	Version   string   `json:"version"`
}

func (d *GroupsClient) List() ([]Group, error) {
	r, err := d.client.Get(d.URL(""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %s", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get database groups: received status code%w", parseResponseError(r))
	}

	type ListResponse struct {
		Groups []Group `json:"groups"`
	}
	resp, err := unmarshal[ListResponse](r)
	return resp.Groups, err
}

func (d *GroupsClient) Get(name string) (Group, error) {
	r, err := d.client.Get(d.URL("/"+name), nil)
	if err != nil {
		return Group{}, fmt.Errorf("failed to get group %s: %w", name, err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return Group{}, notMemberErr(org)
	}

	if r.StatusCode == http.StatusNotFound {
		return Group{}, fmt.Errorf("group %s was not found", name)
	}

	if r.StatusCode != http.StatusOK {
		return Group{}, fmt.Errorf("failed to get database group: received status code%w", parseResponseError(r))
	}

	type Response struct {
		Group Group `json:"group"`
	}
	resp, err := unmarshal[Response](r)
	return resp.Group, err
}

func (d *GroupsClient) Delete(group string) error {
	url := d.URL("/" + group)
	r, err := d.client.Delete(url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete group: %s", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode == http.StatusNotFound {
		return fmt.Errorf("group %s not found. List known databases using %s", internal.Emph(group), internal.Emph("turso group list"))
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete group: received status code%w", parseResponseError(r))
	}

	return nil
}

func (d *GroupsClient) Create(name, location, version string) error {
	type Body struct{ Name, Location, Version string }
	body, err := marshal(Body{name, location, version})
	if err != nil {
		return fmt.Errorf("could not serialize request body: %w", err)
	}

	res, err := d.client.Post(d.URL(""), body)
	if err != nil {
		return fmt.Errorf("failed to create group: %s", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return notMemberErr(org)
	}

	if res.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("group name '%s' is not available", name)
	}

	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	return nil
}

func (d *GroupsClient) Unarchive(name string) error {
	res, err := d.client.Post(d.URL("/"+name+"/unarchive"), nil)
	if err != nil {
		return fmt.Errorf("failed to unarchive group: %s", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return notMemberErr(org)
	}

	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	return nil
}

func (d *GroupsClient) AddLocation(name, location string) error {
	res, err := d.client.Post(d.URL("/"+name+"/locations/"+location), nil)
	if err != nil {
		return fmt.Errorf("failed to post group location request: %s", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return notMemberErr(org)
	}

	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	return nil
}

func (d *GroupsClient) RemoveLocation(name, location string) error {
	res, err := d.client.Delete(d.URL("/"+name+"/locations/"+location), nil)
	if err != nil {
		return fmt.Errorf("failed to post group location request: %s", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return notMemberErr(org)
	}

	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	return nil
}

func (d *GroupsClient) WaitLocation(name, location string) error {
	res, err := d.client.Get(d.URL("/"+name+"/locations/"+location+"/wait"), nil)
	if err != nil {
		return fmt.Errorf("failed to send wait location request: %s", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return notMemberErr(org)
	}

	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	return nil
}

type Entities struct {
	DBNames []string `json:"databases,omitempty"`
}

type PermissionsClaim struct {
	ReadAttach Entities `json:"read_attach,omitempty"`
}

type GroupTokenRequest struct {
	Permissions *PermissionsClaim `json:"permissions,omitempty"`
}

func (d *GroupsClient) Token(group string, expiration string, readOnly bool, permissions *PermissionsClaim) (string, error) {
	authorization := ""
	if readOnly {
		authorization = "&authorization=read-only"
	}
	url := d.URL(fmt.Sprintf("/%s/auth/tokens?expiration=%s%s", group, expiration, authorization))

	req := GroupTokenRequest{permissions}
	body, err := marshal(req)
	if err != nil {
		return "", fmt.Errorf("could not serialize request body: %w", err)
	}

	r, err := d.client.Post(url, body)
	if err != nil {
		return "", fmt.Errorf("failed to get database token: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return "", notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get database token: %w", parseResponseError(r))
	}

	type JwtResponse struct{ Jwt string }
	data, err := unmarshal[JwtResponse](r)
	if err != nil {
		return "", err
	}
	return data.Jwt, nil
}

func (d *GroupsClient) Rotate(group string) error {
	url := d.URL(fmt.Sprintf("/%s/auth/rotate", group))
	r, err := d.client.Post(url, nil)
	if err != nil {
		return fmt.Errorf("failed to rotate database keys: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to rotate database keys: %w", parseResponseError(r))
	}

	return nil
}

func (d *GroupsClient) Update(group string, version, extensions string) error {
	type Body struct{ Version, Extensions string }
	body, err := marshal(Body{version, extensions})
	if err != nil {
		return fmt.Errorf("could not serialize request body: %w", err)
	}

	url := d.URL(fmt.Sprintf("/%s/update", group))
	r, err := d.client.Post(url, body)
	if err != nil {
		return fmt.Errorf("failed to rotate database keys: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update group: %w", parseResponseError(r))
	}

	return nil
}

func (d *GroupsClient) Transfer(group string, to string) error {
	type Body struct {
		Organization string `json:"organization"`
	}
	body, err := marshal(Body{to})
	if err != nil {
		return fmt.Errorf("could not serialize request body: %w", err)
	}

	url := d.URL(fmt.Sprintf("/%s/transfer", group))
	r, err := d.client.Post(url, body)
	if err != nil {
		return fmt.Errorf("failed to transfer group: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		err := parseResponseError(r)
		return fmt.Errorf("failed to transfer group: %w", err)
	}

	return nil
}

func (d *GroupsClient) URL(suffix string) string {
	prefix := "/v1"
	if d.client.Org != "" {
		prefix = "/v1/organizations/" + d.client.Org
	}
	return prefix + "/groups" + suffix
}
