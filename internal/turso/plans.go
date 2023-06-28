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

type OrgPlan struct {
	Active    string `json:"active"`
	Scheduled string `json:"scheduled"`
}

func (c *PlansClient) Get() (OrgPlan, error) {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}

	r, err := c.client.Get(prefix+"/plan", nil)
	if err != nil {
		return OrgPlan{}, fmt.Errorf("failed to get organization plan: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return OrgPlan{}, fmt.Errorf("failed to get organization plan with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Plan OrgPlan }](r)
	return resp.Plan, err
}

var ErrPaymentRequired = errors.New("payment required")

func (c *PlansClient) Set(plan string) (OrgPlan, error) {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}

	body, err := marshal(struct {
		Plan string `json:"plan"`
	}{plan})
	if err != nil {
		return OrgPlan{}, fmt.Errorf("could not serialize request body: %w", err)
	}

	r, err := c.client.Post(prefix+"/plan", body)
	if err != nil {
		return OrgPlan{}, fmt.Errorf("failed to set organization plan: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusPaymentRequired {
		return OrgPlan{}, ErrPaymentRequired
	}

	if r.StatusCode != 200 {
		return OrgPlan{}, fmt.Errorf("failed to set organization plan with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Plan OrgPlan }](r)
	return resp.Plan, err
}
