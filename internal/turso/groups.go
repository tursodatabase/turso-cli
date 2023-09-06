package turso

import (
	"fmt"
	"net/http"

	"github.com/chiselstrike/iku-turso-cli/internal"
)

type GroupsClient client

type Group struct {
	Name      string   `json:"name"`
	Locations []string `json:"locations"`
	Primary   string   `json:"primary"`
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
		return nil, fmt.Errorf("failed to get database groups: received status code %s", r.Status)
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
		return Group{}, fmt.Errorf("failed to get database group: received status code %s", r.Status)
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
		return fmt.Errorf("failed to delete group: received status code %s", r.Status)
	}

	return nil
}

func (d *GroupsClient) Create(name, location string) error {
	type Body struct{ Name, Location string }
	body, err := marshal(Body{name, location})
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

func (d *GroupsClient) URL(suffix string) string {
	prefix := "/v1"
	if d.client.Org != "" {
		prefix = "/v1/organizations/" + d.client.Org
	}
	return prefix + "/groups" + suffix
}
