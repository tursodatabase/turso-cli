package turso

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/chiselstrike/iku-turso-cli/internal"
)

type Database struct {
	ID            string `json:"dbId" mapstructure:"dbId"`
	Name          string
	Regions       []string
	PrimaryRegion string
	Hostname      string
}

type DatabasesClient client

type CreateInstanceLocationError struct {
	err string
}

func (e *CreateInstanceLocationError) Error() string {
	return e.err
}

func (d *DatabasesClient) List() ([]Database, error) {
	r, err := d.client.Get(d.URL(""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get database listing: %s", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get database listing: %s", r.Status)
	}

	type ListResponse struct {
		Databases []Database `json:"databases"`
	}
	resp, err := unmarshal[ListResponse](r)
	return resp.Databases, err
}

func (d *DatabasesClient) Delete(database string) error {
	url := d.URL("/" + database)
	r, err := d.client.Delete(url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete database: %s", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode == http.StatusNotFound {
		return fmt.Errorf("database %s not found. List known databases using %s", internal.Emph(database), internal.Emph("turso db list"))
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete database: %s", r.Status)
	}

	return nil
}

type CreateDatabaseResponse struct {
	Database Database
	Username string
}

func (d *DatabasesClient) Create(name, region, image, extensions string) (*CreateDatabaseResponse, error) {
	type Body struct{ Name, Region, Image, Extensions string }
	body, err := marshal(Body{name, region, image, extensions})
	if err != nil {
		return nil, fmt.Errorf("could not serialize request body: %w", err)
	}

	res, err := d.client.Post(d.URL(""), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %s", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if res.StatusCode == http.StatusUnprocessableEntity {
		return nil, fmt.Errorf("database name '%s' is not available", name)
	}

	if res.StatusCode != http.StatusOK {
		return nil, parseResponseError(res)
	}

	data, err := unmarshal[*CreateDatabaseResponse](res)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return data, nil
}

func (d *DatabasesClient) Seed(name string, dbFile *os.File) error {
	url := d.URL(fmt.Sprintf("/%s/seed", name))
	res, err := d.client.Upload(url, dbFile)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return notMemberErr(org)
	}

	if res.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("database name '%s' is not available", name)
	}

	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	return nil
}

func (d *DatabasesClient) Token(database string, expiration string, readOnly bool) (string, error) {
	authorization := ""
	if readOnly {
		authorization = "&authorization=read-only"
	}
	url := d.URL(fmt.Sprintf("/%s/auth/tokens?expiration=%s%s", database, expiration, authorization))
	r, err := d.client.Post(url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get database token: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return "", notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		err, _ := unmarshal[string](r)
		return "", fmt.Errorf("failed to get database token: %d %s", r.StatusCode, err)
	}

	type JwtResponse struct{ Jwt string }
	data, err := unmarshal[JwtResponse](r)
	if err != nil {
		return "", err
	}
	return data.Jwt, nil
}

func (d *DatabasesClient) Rotate(database string) error {
	url := d.URL(fmt.Sprintf("/%s/auth/rotate", database))
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
		err, _ := unmarshal[string](r)
		return fmt.Errorf("failed to rotate database keys: %d %s", r.StatusCode, err)
	}

	return nil
}

func (d *DatabasesClient) Update(database string) error {
	url := d.URL(fmt.Sprintf("/%s/update", database))
	r, err := d.client.Post(url, nil)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		err, _ := unmarshal[string](r)
		return fmt.Errorf("failed to update database: %d %s", r.StatusCode, err)
	}

	return nil
}

type Body struct {
	Org string `json:"org"`
}

func (d *DatabasesClient) Transfer(database, org string) error {
	url := d.URL(fmt.Sprintf("/%s/transfer", database))
	body, err := json.Marshal(Body{Org: org})
	bodyReader := bytes.NewReader(body)
	if err != nil {
		return fmt.Errorf("could not serialize request body: %w", err)
	}
	r, err := d.client.Post(url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to transfer database")
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusForbidden {
		return fmt.Errorf("user is not an owner of the organizations involved in the transfer.")
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to transfer database to org %s.", org)
	}

	return nil
}

type Usage struct {
	RowsRead         uint64 `json:"rows_read,omitempty"`
	RowsWritten      uint64 `json:"rows_written,omitempty"`
	StorageBytesUsed uint64 `json:"storage_bytes,omitempty"`
}

type DbUsage struct {
	Instances map[string]Usage `json:"instances"`
	Total     Usage            `json:"total"`
}

func (d *DatabasesClient) Usage(database string) (DbUsage, error) {
	url := d.URL(fmt.Sprintf("/%s/usage", database))

	r, err := d.client.Get(url, nil)
	if err != nil {
		return DbUsage{}, fmt.Errorf("failed to get database usage: %w", err)
	}
	defer r.Body.Close()

	body, err := unmarshal[DbUsage](r)
	return body, err
}

func (d *DatabasesClient) URL(suffix string) string {
	prefix := "/v1"
	if d.client.Org != "" {
		prefix = "/v1/organizations/" + d.client.Org
	}
	return prefix + "/databases" + suffix
}
