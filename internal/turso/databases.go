package turso

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
)

const multipartUploadThresholdBytes = 100 * 1024 * 1024 // 100MB

type Database struct {
	ID               string `json:"dbId" mapstructure:"dbId"`
	Name             string
	Regions          []string
	PrimaryRegion    string
	Hostname         string
	Version          string
	Group            string
	Sleeping         bool
	Schema           string
	IsSchema         bool      `json:"is_schema" mapstructure:"is_schema"`
	Parent           *Database `json:"parent,omitempty"`
	EncryptionCipher string    `json:"encryption_cipher,omitempty"`
}

type DatabasesClient client

type DatabaseListOptions struct {
	Group  string
	Schema string
	Limit  int
	Cursor string
	Parent string
}

func (o DatabaseListOptions) Encode() string {
	query := url.Values{}
	if o.Group != "" {
		query.Set("group", o.Group)
	}
	if o.Schema != "" {
		query.Set("schema", o.Schema)
	}
	if o.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Cursor != "" {
		query.Set("cursor", o.Cursor)
	}
	if o.Parent != "" {
		query.Set("parent", o.Parent)
	}
	return query.Encode()
}

type ListResponse struct {
	Databases  []Database  `json:"databases"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

func (d *DatabasesClient) List(options DatabaseListOptions) (ListResponse, error) {
	path := d.URL("")

	if options := options.Encode(); options != "" {
		path += "?" + options
	}

	r, err := d.client.Get(path, nil)
	if err != nil {
		return ListResponse{}, fmt.Errorf("failed to get database listing: %s", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return ListResponse{}, notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return ListResponse{}, fmt.Errorf("failed to get database listing: %w", parseResponseError(r))
	}

	resp, err := unmarshal[ListResponse](r)
	if err != nil {
		return ListResponse{}, err
	}

	return resp, nil
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
	// This is only used locally when uploading a database file and
	// never passed to the control plane as JSON.
	Filepath  string `json:"-"`
	Multipart bool   `json:"-"`
}

type RemoteEncryption struct {
	EncryptionKey    string `json:"encryption_key"`
	EncryptionCipher string `json:"encryption_cipher"`
}

type CreateDatabaseBody struct {
	Name             string            `json:"name"`
	Location         string            `json:"location"`
	Image            string            `json:"image,omitempty"`
	Extensions       string            `json:"extensions,omitempty"`
	Group            string            `json:"group,omitempty"`
	Seed             *DBSeed           `json:"seed,omitempty"`
	Schema           string            `json:"schema,omitempty"`
	IsSchema         bool              `json:"is_schema,omitempty"`
	SizeLimit        string            `json:"size_limit,omitempty"`
	RemoteEncryption *RemoteEncryption `json:"remote_encryption,omitempty"`
}

func (d *DatabasesClient) Create(name, location, image, extensions, group string, schema string, isSchema bool, seed *DBSeed, sizeLimit, remoteEncryptionCipher, remoteEncryptionKey string, spinner *prompt.SpinnerT) (*CreateDatabaseResponse, error) {
	isTursoServerUpload := seed != nil && seed.Type == "database_upload" && seed.Filepath != ""
	var uploadFilepath string
	var useMultipart bool
	var params CreateDatabaseBody
	if isTursoServerUpload {
		uploadFilepath = seed.Filepath
		useMultipart = seed.Multipart
		// Clear the unused seed parameters, only Type=database_upload is used.
		seed.Filepath = ""
		seed.Name = ""
		seed.URL = ""
		seed.Timestamp = nil
		seed.Multipart = false
		params = CreateDatabaseBody{
			Name:     name,
			Location: location,
			Group:    group,
			Seed:     seed,
		}
	} else {
		params = CreateDatabaseBody{name, location, image, extensions, group, seed, schema, isSchema, sizeLimit, nil}
	}

	if remoteEncryptionKey != "" {
		params.RemoteEncryption = &RemoteEncryption{
			EncryptionKey:    remoteEncryptionKey,
			EncryptionCipher: remoteEncryptionCipher,
		}
	}

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

	if isTursoServerUpload {
		if _, err = d.UploadDatabaseAWS(data, group, uploadFilepath, remoteEncryptionCipher, remoteEncryptionKey, useMultipart, spinner); err != nil {
			// Clean up the database if the upload fails
			if deleteErr := d.Delete(data.Database.Name); deleteErr != nil {
				fmt.Printf("%v", deleteErr)
			}
			return nil, err
		}

		return data, nil
	}

	return data, nil
}

// UploadDatabaseAWS creates a database from an uploaded database file
//  1. It creates a database on the control plane as normal, but passes a special seed type
//     which instructs the control plane create the db as 'draft',
//     i.e. in a mode where it is not yet available for use.
//     This call happens in DatabasesClient.Create() above, after which it calls this function.
//  2. This function creates a DB token for the newly-created DB, and then calls turso-server to upload the database file.
//     turso-server will perform validations on the file and 'activate' the db if everything is ok.
func (d *DatabasesClient) UploadDatabaseAWS(resp *CreateDatabaseResponse, group, uploadFilepath, remoteEncryptionCipher, remoteEncryptionKey string, useMultipart bool, spinner *prompt.SpinnerT) (*CreateDatabaseResponse, error) {
	// Create a short-lived DB token for the newly created database to facilitate the upload
	token, err := d.Token(resp.Database.Name, "1h", false, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create database token: %w", err)
	}

	baseURL, err := url.Parse(fmt.Sprintf("https://%s", resp.Database.Hostname))
	if err != nil {
		return nil, fmt.Errorf("unable to create TursoServerClient: %v", err)
	}
	tursoServerClient, err := NewTursoServerClient(baseURL, token, d.client.cliVersion, d.client.Org)
	if err != nil {
		return nil, fmt.Errorf("could not create Turso server client: %w", err)
	}

	// Upload the database file
	spinner.Text(fmt.Sprintf("Uploading database %s in group %s, this may take a while...", internal.Emph(resp.Database.Name), internal.Emph(group)))

	stat, err := os.Stat(uploadFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file size %s: %w", uploadFilepath, err)
	}

	uploadFunc := tursoServerClient.UploadFileSinglePart
	if useMultipart || stat.Size() > multipartUploadThresholdBytes {
		uploadFunc = tursoServerClient.UploadFileMultipart
	}

	err = uploadFunc(uploadFilepath, remoteEncryptionCipher, remoteEncryptionKey, func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool) {
		totalSeconds := int(elapsedTime.Seconds())
		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		secondsStr := "seconds"
		if seconds == 1 {
			secondsStr = "second"
		}
		minutesStr := "minutes"
		if minutes == 1 {
			minutesStr = "minute"
		}
		var elapsedTimeStr string
		if minutes > 0 {
			elapsedTimeStr = fmt.Sprintf("%d %s %d %s", minutes, minutesStr, seconds, secondsStr)
		} else {
			elapsedTimeStr = fmt.Sprintf("%d %s", seconds, secondsStr)
		}
		if done {
			spinner.Text(fmt.Sprintf("Uploaded database %s in group %s (%d bytes) - we are now verifying your database on the server... (took %s)", internal.Emph(resp.Database.Name), internal.Emph(group), totalBytes, elapsedTimeStr))
		} else {
			spinner.Text(fmt.Sprintf("Uploading database %s in group %s, %d%% complete (%d/%d bytes uploaded) (elapsed %s)", internal.Emph(resp.Database.Name), internal.Emph(group), progressPct, uploadedBytes, totalBytes, elapsedTimeStr))
		}
	})
	if err != nil {
		return nil, fmt.Errorf("could not upload database file: %w", err)
	}

	// Return the original database creation response
	return resp, nil
}

func (d *DatabasesClient) Export(dbName, dbUrl, outputFile string, withMetadata bool, overwrite bool, remoteEncryptionKey string) error {
	if !overwrite {
		if _, err := os.Stat(outputFile); err == nil {
			return fmt.Errorf("file %s already exists, use `--overwrite` flag to overwrite it", outputFile)
		}
	}
	token, err := d.Token(dbName, "1h", false, nil, nil)
	if err != nil {
		return fmt.Errorf("could not create database token: %w", err)
	}
	baseURL, err := url.Parse(dbUrl)
	if err != nil {
		return fmt.Errorf("could not parse database URL: %w", err)
	}
	tursoServerClient, err := NewTursoServerClient(baseURL, token, d.client.cliVersion, d.client.Org)
	if err != nil {
		return fmt.Errorf("could not create Turso server client: %w", err)
	}
	return tursoServerClient.Export(outputFile, withMetadata, remoteEncryptionKey)
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

type DatabaseTokenRequest struct {
	Permissions            *PermissionsClaim              `json:"permissions,omitempty"`
	FineGrainedPermissions []flags.FineGrainedPermissions `json:"fine_grained_permissions,omitempty"`
}

func (d *DatabasesClient) Token(
	database string,
	expiration string,
	readOnly bool,
	permissions *PermissionsClaim,
	fineGrainedPermissions []flags.FineGrainedPermissions,
) (string, error) {
	authorization := ""
	if readOnly {
		authorization = "&authorization=read-only"
	}
	url := d.URL(fmt.Sprintf("/%s/auth/tokens?expiration=%s%s", database, expiration, authorization))

	req := DatabaseTokenRequest{Permissions: permissions, FineGrainedPermissions: fineGrainedPermissions}
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
		return fmt.Errorf("failed to rotate database keys: %w", parseResponseError(r))
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
		return fmt.Errorf("failed to update database: %w", parseResponseError(r))
	}

	return nil
}

type Stats struct {
	Query       string `json:"query"`
	RowsRead    int    `json:"rows_read"`
	RowsWritten int    `json:"rows_written"`
}

func (d *DatabasesClient) Stats(database string) ([]Stats, error) {
	from := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	url := d.URL(fmt.Sprintf("/%s/usage/queries?from=%v", database, from))
	r, err := d.client.Get(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update database: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get stats for database: %w", parseResponseError(r))
	}

	return unmarshal[[]Stats](r)
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
		return errors.New("failed to transfer database")
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to transfer %s database to org %s: %w", database, org, parseResponseError(r))
	}

	return nil
}

func (d *DatabasesClient) Wakeup(database string) error {
	url := d.URL(fmt.Sprintf("/%s/wakeup", database))
	r, err := d.client.Post(url, nil)
	if err != nil {
		return fmt.Errorf("failed to unarchive database: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to unarchive database: %w", parseResponseError(r))
	}

	return nil
}

type Usage struct {
	RowsRead         uint64 `json:"rows_read,omitempty"`
	RowsWritten      uint64 `json:"rows_written,omitempty"`
	StorageBytesUsed uint64 `json:"storage_bytes,omitempty"`
	BytesSynced      uint64 `json:"bytes_synced,omitempty"`
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
		return DbUsage{}, fmt.Errorf("failed to get database usage: %w", parseResponseError(r))
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

type Pagination struct {
	Next *string `json:"next"`
}

type DatabaseConfig struct {
	AllowAttach      *bool `json:"allow_attach"`
	DeleteProtection *bool `json:"delete_protection"`
}

func (d *DatabaseConfig) IsDeleteProtected() bool {
	if d.DeleteProtection == nil {
		return false
	}
	return *d.DeleteProtection
}

func (d *DatabaseConfig) AttachAllowed() bool {
	if d.AllowAttach == nil {
		return false
	}
	return *d.AllowAttach
}

func (d *DatabasesClient) GetConfig(database string) (DatabaseConfig, error) {
	url := d.URL(fmt.Sprintf("/%s/configuration", database))
	r, err := d.client.Get(url, nil)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("failed to get database: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return DatabaseConfig{}, notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		err = parseResponseError(r)
		return DatabaseConfig{}, fmt.Errorf("failed to get config for database: %d %s", r.StatusCode, err)
	}

	return unmarshal[DatabaseConfig](r)
}

func (d *DatabasesClient) UpdateConfig(database string, config DatabaseConfig) error {
	url := d.URL(fmt.Sprintf("/%s/configuration", database))
	body, err := marshal(config)
	if err != nil {
		return fmt.Errorf("could not serialize request body: %w", err)
	}
	r, err := d.client.Patch(url, body)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	defer r.Body.Close()

	org := d.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		err = parseResponseError(r)
		return fmt.Errorf("failed to update config for database: %d %s", r.StatusCode, err)
	}

	return nil
}
