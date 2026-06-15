package turso

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/tursodatabase/turso-cli/internal/flags"
)

type GroupsV3Client client

func (g *GroupsV3Client) url(orgID, suffix string) string {
	return "/v3/organizations/" + orgID + "/groups" + suffix
}

func (g *GroupsV3Client) List(orgID string) ([]Group, error) {
	r, err := g.client.Get(g.url(orgID, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list groups: %w", parseResponseError(r))
	}

	type response struct {
		Groups []Group `json:"groups"`
	}
	resp, err := unmarshal[response](r)
	if err != nil {
		return nil, err
	}
	return resp.Groups, nil
}

func (g *GroupsV3Client) Get(orgID, groupID string) (Group, error) {
	r, err := g.client.Get(g.url(orgID, "/"+groupID), nil)
	if err != nil {
		return Group{}, fmt.Errorf("failed to get group: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return Group{}, fmt.Errorf("failed to get group: %w", parseResponseError(r))
	}

	type response struct {
		Group Group `json:"group"`
	}
	resp, err := unmarshal[response](r)
	if err != nil {
		return Group{}, err
	}
	return resp.Group, nil
}

func (g *GroupsV3Client) Token(
	orgID, groupID string,
	expiration string,
	readOnly bool,
	fineGrainedPermissions []flags.FineGrainedPermissions,
) (string, error) {
	q := url.Values{}
	if expiration != "" {
		q.Set("expiration", expiration)
	}
	if readOnly {
		q.Set("authorization", "read-only")
	}
	path := g.url(orgID, "/"+groupID+"/auth/tokens")
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	req := GroupTokenRequest{FineGrainedPermissions: fineGrainedPermissions}
	body, err := marshal(req)
	if err != nil {
		return "", fmt.Errorf("could not serialize request body: %w", err)
	}
	r, err := g.client.Post(path, body)
	if err != nil {
		return "", fmt.Errorf("failed to get group token: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get group token: %w", parseResponseError(r))
	}

	type response struct {
		Jwt string `json:"jwt"`
	}
	resp, err := unmarshal[response](r)
	if err != nil {
		return "", err
	}
	return resp.Jwt, nil
}
