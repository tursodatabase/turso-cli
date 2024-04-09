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
		BytesSynced uint64 `json:"bytesSynced"`
		Locations   uint64 `json:"locations"`
		Storage     uint64 `json:"storage"`
		Groups      uint64 `json:"groups"`
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

type Subscription struct {
	Plan     string `json:"plan"`
	Timeline string `json:"timeline"`
	Overages bool   `json:"overages"`
}

func (c *SubscriptionClient) Get() (Subscription, error) {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}

	r, err := c.client.Get(prefix+"/subscription", nil)
	if err != nil {
		return Subscription{}, fmt.Errorf("failed to get organization plan: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return Subscription{}, fmt.Errorf("failed to get organization plan with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Subscription Subscription }](r)
	return resp.Subscription, err
}

var ErrPaymentRequired = errors.New("payment required")

func (c *SubscriptionClient) Update(plan, timeline string, overages *bool) error {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}

	body, err := marshal(struct {
		Plan     string `json:"plan"`
		Timeline string `json:"timeline,omitempty"`
		Overages *bool  `json:"overages,omitempty"`
	}{plan, timeline, overages})
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
