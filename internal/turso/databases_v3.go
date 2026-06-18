package turso

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/tursodatabase/turso-cli/internal/flags"
)

type DatabasesV3Client client

// CreateDatabaseV3Body mirrors the legacy create request (turso.CreateDBParams):
// forks are expressed through Seed just like v1/v2. The single v3-specific
// addition is GroupID, which carries the group UUID instead of the group name.
type CreateDatabaseV3Body struct {
	Name             string            `json:"name"`
	GroupID          string            `json:"group_id,omitempty"`
	Seed             *DBSeed           `json:"seed,omitempty"`
	RemoteEncryption *RemoteEncryption `json:"remote_encryption,omitempty"`
	UseTursoDB       bool              `json:"use_tursodb,omitempty"`
}

func (d *DatabasesV3Client) url(orgID, suffix string) string {
	return "/v3/organizations/" + orgID + "/databases" + suffix
}

type DatabaseV3ListOptions struct {
	GroupId    string
	ParentDbId string
	Limit      int
	Cursor     string
}

func (d *DatabasesV3Client) List(orgID string, options DatabaseV3ListOptions) ([]Database, string, error) {
	path := d.url(orgID, "")
	q := url.Values{}
	if options.GroupId != "" {
		q.Set("group_id", options.GroupId)
	}
	if options.ParentDbId != "" {
		q.Set("parent_db_id", options.ParentDbId)
	}
	if options.Limit != 0 {
		q.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.Cursor != "" {
		q.Set("cursor", options.Cursor)
	}
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	r, err := d.client.Get(path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list databases: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to list databases: %w", parseResponseError(r))
	}

	type response struct {
		Databases  []Database  `json:"databases"`
		Pagination *Pagination `json:"pagination,omitempty"`
	}
	resp, err := unmarshal[response](r)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if resp.Pagination != nil && resp.Pagination.Next != nil {
		next = *resp.Pagination.Next
	}
	return resp.Databases, next, nil
}

func (d *DatabasesV3Client) Get(orgID, dbID string) (Database, error) {
	r, err := d.client.Get(d.url(orgID, "/"+dbID), nil)
	if err != nil {
		return Database{}, fmt.Errorf("failed to get database: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return Database{}, fmt.Errorf("failed to get database: %w", parseResponseError(r))
	}

	type response struct {
		Database Database `json:"database"`
	}
	resp, err := unmarshal[response](r)
	if err != nil {
		return Database{}, err
	}
	return resp.Database, nil
}

func (d *DatabasesV3Client) GetConfig(orgID, dbID string) (DatabaseConfig, error) {
	r, err := d.client.Get(d.url(orgID, "/"+dbID), nil)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("failed to get database: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return DatabaseConfig{}, fmt.Errorf("failed to get database: %w", parseResponseError(r))
	}

	// v3 has no dedicated /configuration endpoint: the database response
	// carries delete protection directly and the allow rules nested under
	// allow_rules_config, so we project them onto DatabaseConfig here.
	type response struct {
		Database struct {
			DeleteProtection bool `json:"delete_protection"`
			AllowRulesConfig struct {
				AllowedIPs       []string `json:"allowed_ips"`
				AllowedAwsVpcIDs []string `json:"allowed_aws_vpc_ids"`
			} `json:"allow_rules_config"`
		} `json:"database"`
	}
	resp, err := unmarshal[response](r)
	if err != nil {
		return DatabaseConfig{}, err
	}
	deleteProtection := resp.Database.DeleteProtection
	config := DatabaseConfig{DeleteProtection: &deleteProtection}
	if ips := resp.Database.AllowRulesConfig.AllowedIPs; ips != nil {
		config.AllowedIPs = &ips
	}
	if vpcs := resp.Database.AllowRulesConfig.AllowedAwsVpcIDs; vpcs != nil {
		config.AllowedAwsVpcIDs = &vpcs
	}
	return config, nil
}

func (d *DatabasesV3Client) Delete(orgID, dbID string) error {
	r, err := d.client.Delete(d.url(orgID, "/"+dbID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete database: %w", parseResponseError(r))
	}
	return nil
}

func (d *DatabasesV3Client) Create(orgID string, body CreateDatabaseV3Body) (Database, error) {
	payload, err := marshal(body)
	if err != nil {
		return Database{}, fmt.Errorf("could not serialize request body: %w", err)
	}

	r, err := d.client.Post(d.url(orgID, ""), payload)
	if err != nil {
		return Database{}, fmt.Errorf("failed to create database: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return Database{}, fmt.Errorf("failed to create database: %w", parseResponseError(r))
	}

	type response struct {
		Database Database `json:"database"`
	}
	resp, err := unmarshal[response](r)
	if err != nil {
		return Database{}, err
	}
	return resp.Database, nil
}

func (d *DatabasesV3Client) Token(
	orgID string, dbID string,
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
	path := d.url(orgID, "/"+dbID+"/auth/tokens")
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	req := DatabaseTokenRequest{FineGrainedPermissions: fineGrainedPermissions}
	body, err := marshal(req)
	if err != nil {
		return "", fmt.Errorf("could not serialize request body: %w", err)
	}
	r, err := d.client.Post(path, body)
	if err != nil {
		return "", fmt.Errorf("failed to get database token: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get database token: %w", parseResponseError(r))
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
