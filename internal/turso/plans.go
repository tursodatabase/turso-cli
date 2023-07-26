package turso

import (
	"errors"
	"fmt"
	"net/http"
)

type PlansClient client

type Plan struct {
	Name   string `json:"name"`
	Price  string `json:"price"`
	Quotas struct {
		RowsRead    uint64 `json:"rowsRead"`
		RowsWritten uint64 `json:"rowsWritten"`
		Databases   uint64 `json:"databases"`
		Locations   uint64 `json:"locations"`
		Storage     uint64 `json:"storage"`
	}
}

func (c *PlansClient) List() ([]Plan, error) {
	r, err := c.client.Get("/v1/plans", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan list: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list plans with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Plans []Plan }](r)
	return resp.Plans, err
}

type SubscriptionClient client

func (c *SubscriptionClient) Get() (string, error) {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}

	r, err := c.client.Get(prefix+"/subscription", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get organization plan: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return "", fmt.Errorf("failed to get organization plan with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Subscription struct{ Name string } }](r)
	return resp.Subscription.Name, err
}

var ErrPaymentRequired = errors.New("payment required")

func (c *SubscriptionClient) Set(plan string) error {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}

	body, err := marshal(struct {
		Plan string `json:"plan"`
	}{plan})
	if err != nil {
		return fmt.Errorf("could not serialize request body: %w", err)
	}

	r, err := c.client.Post(prefix+"/subscription", body)
	if err != nil {
		return fmt.Errorf("failed to set organization plan: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusPaymentRequired {
		return ErrPaymentRequired
	}

	if r.StatusCode != 200 {
		return fmt.Errorf("failed to set organization plan with status %s: %v", r.Status, parseResponseError(r))
	}

	return nil
}
