package turso

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type RestoreState string

var (
	RestoreStateStarting    RestoreState = "starting"
	RestoreStateStarted     RestoreState = "started"
	RestoreStateRestoring   RestoreState = "restoring"
	RestoreStateRestored    RestoreState = "restored"
	RestoreStateDownloading RestoreState = "downloading"
	RestoreStateDownloaded  RestoreState = "downloaded"
	RestoreStateFailed      RestoreState = "failed"
)

type Restore struct {
	ID uint `gorm:"primarykey" json:"id"`

	Timestamp  time.Time
	DatabaseID uint

	State              RestoreState `json:"state"`
	StateBeforeFailure RestoreState `json:"state_before_failure,omitempty"`
}

type RestoreClient client

func (rc *RestoreClient) Create(db string, timestamp *time.Time) (*Restore, error) {
	type Body struct {
		Timestamp *time.Time `json:"timestamp,omitempty"`
	}
	body, err := marshal(Body{Timestamp: timestamp})
	if err != nil {
		return nil, fmt.Errorf("failed to marshall create restore request body: %s", err)
	}

	resp, err := rc.client.Post(rc.URL(db, ""), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create restore for database %s: %s", db, err)
	}
	defer resp.Body.Close()

	org := rc.client.org
	if isNotMemberErr(resp.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseResponseError(resp)
	}

	respBody, err := unmarshal[Restore](resp)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return &respBody, nil
}

func (rc *RestoreClient) Get(db string, restoreId uint) (*Restore, error) {
	resp, err := rc.client.Get(rc.URL(db, fmt.Sprintf("%d", restoreId)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get restore %d from database %s: %s", restoreId, db, err)
	}
	defer resp.Body.Close()

	org := rc.client.org
	if isNotMemberErr(resp.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseResponseError(resp)
	}

	respBody, err := unmarshal[Restore](resp)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (rc *RestoreClient) Download(db string, restoreId uint) (io.ReadCloser, error) {
	resp, err := rc.client.Get(rc.URL(db, fmt.Sprintf("%d/download", restoreId)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get restore %d from database %s: %s", restoreId, db, err)
	}

	org := rc.client.org
	if isNotMemberErr(resp.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseResponseError(resp)
	}

	return resp.Body, nil
}

func (rc *RestoreClient) URL(database, restore string) string {
	url := "/v1"
	if rc.client.org != "" {
		url = "/v1/organizations/" + rc.client.org
	}
	url = fmt.Sprintf("%s/databases/%s/restore", url, database)
	if restore != "" {
		url = fmt.Sprintf("%s/%s", url, restore)
	}
	return url
}
