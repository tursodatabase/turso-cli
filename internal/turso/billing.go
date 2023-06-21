package turso

import "fmt"

type BillingClient client

type Portal struct {
	URL string `json:"url"`
}

func (c *BillingClient) Portal() (Portal, error) {
	prefix := "/v1"
	if c.client.org != "" {
		prefix = "/v1/organizations/" + c.client.org
	}

	r, err := c.client.Post(prefix+"/billing/portal", nil)
	if err != nil {
		return Portal{}, fmt.Errorf("failed to get database usage: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return Portal{}, fmt.Errorf("failed to get billing portal with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Portal Portal }](r)
	return resp.Portal, err
}

func (c *BillingClient) HasPaymentMethod() (bool, error) {
	prefix := "/v1"
	if c.client.org != "" {
		prefix = "/v1/organizations/" + c.client.org
	}
	r, err := c.client.Get(prefix+"/billing/payment-methods", nil)
	if err != nil {
		return false, fmt.Errorf("failed to get database usage: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return false, fmt.Errorf("failed to check payment method with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Exists bool }](r)
	return resp.Exists, err
}
