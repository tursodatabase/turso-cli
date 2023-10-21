package turso

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/tursodatabase/turso-cli/internal"
)

type Database struct {
	ID            string `json:"dbId" mapstructure:"dbId"`
	Name          string
	Regions       []string
	PrimaryRegion string
	Hostname      string
	Version       string
	Group         string
}

type DatabasesClient client

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
		return nil, fmt.Errorf("failed to get database listing: %w", parseResponseError(r))
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
		return fmt.Errorf("failed to delete database: %w", parseResponseError(r))
	}

	return nil
}

type CreateDatabaseResponse struct {
	Database Database
	Username string
}

type DBSeed struct {
	Type      string     `json:"type"`
	Name      string     `json:"value,omitempty"`
	URL       string     `json:"url,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

type CreateDatabaseBody struct {
	Name       string  `json:"name"`
	Location   string  `json:"location"`
	Image      string  `json:"image,omitempty"`
	Extensions string  `json:"extensions,omitempty"`
	Group      string  `json:"group,omitempty"`
	Seed       *DBSeed `json:"seed,omitempty"`
}

func (d *DatabasesClient) Create(name, location, image, extensions, group string, seed *DBSeed) (*CreateDatabaseResponse, error) {
	params := CreateDatabaseBody{name, location, image, extensions, group, seed}

	body, err := marshal(params)
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

func (d *DatabasesClient) UploadDump(dbFile *os.File) (string, error) {
	url := d.URL("/dumps")
	res, err := d.client.Upload(url, dbFile)
	if err != nil {
		return "", fmt.Errorf("failed to upload the dump file: %w", err)
	}
	defer res.Body.Close()

	org := d.client.Org
	if isNotMemberErr(res.StatusCode, org) {
		return "", notMemberErr(org)
	}
	if res.StatusCode != http.StatusOK {
		return "", parseResponseError(res)
	}
	type response struct {
		DumpURL string `json:"dump_url"`
	}
	data, err := unmarshal[response](res)
	if err != nil {
		return "", err
	}
	return data.DumpURL, nil
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

func (d *DatabasesClient) Update(database string, group bool) error {
	url := d.URL(fmt.Sprintf("/%s/update", database))
	if group {
		url += "?group=true"
	}
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
		err = parseResponseError(r)
		return fmt.Errorf("%v", err)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to transfer database to org %s", org)
	}

	return nil
}

type Usage struct {
	RowsRead         uint64 `json:"rows_read,omitempty"`
	RowsWritten      uint64 `json:"rows_written,omitempty"`
	StorageBytesUsed uint64 `json:"storage_bytes,omitempty"`
}

type InstanceUsage struct {
	UUID  string `json:"uuid,omitempty"`
	Usage Usage  `json:"usage"`
}

type DbUsage struct {
	UUID      string          `json:"uuid,omitempty"`
	Instances []InstanceUsage `json:"instances"`
	Usage     Usage           `json:"usage"`
}

type DbUsageResponse struct {
	DbUsage DbUsage `json:"database"`
}

func (d *DatabasesClient) Usage(database string) (DbUsage, error) {
	url := d.URL(fmt.Sprintf("/%s/usage", database))

	r, err := d.client.Get(url, nil)
	if err != nil {
		return DbUsage{}, fmt.Errorf("failed to get database usage: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		err, _ := unmarshal[string](r)
		return DbUsage{}, fmt.Errorf("failed to get database usage: %d %s", r.StatusCode, err)
	}

	body, err := unmarshal[DbUsageResponse](r)
	if err != nil {
		return DbUsage{}, err
	}
	return body.DbUsage, nil
}

func (d *DatabasesClient) URL(suffix string) string {
	prefix := "/v1"
	if d.client.Org != "" {
		prefix = "/v1/organizations/" + d.client.Org
	}
	return prefix + "/databases" + suffix
}
