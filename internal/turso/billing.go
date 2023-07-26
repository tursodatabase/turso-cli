package turso

import (
	"fmt"
	"io"
)

type BillingClient client

type Portal struct {
	URL string `json:"url"`
}

func (c *BillingClient) Portal() (Portal, error) {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
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

func (c *BillingClient) PortalForStripeId(stripeId string) (Portal, error) {
	prefix := "/v1"
	type Body struct {
		StripeID string `json:"stripe_id"`
	}
	var body io.Reader
	var err error
	body, err = marshal(Body{StripeID: stripeId})
	if err != nil {
		return Portal{}, fmt.Errorf("could not serialize request body: %w", err)
	}
	r, err := c.client.Post(prefix+"/billing/portal", body)
	if err != nil {
		return Portal{}, fmt.Errorf("failed to get portal: %w", err)
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
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
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

func (c *BillingClient) HasPaymentMethodWithStripeId(stripeId string) (bool, error) {
	prefix := "/v1"
	r, err := c.client.Get(fmt.Sprintf("%s/billing/payment-methods?stripe_id=%s", prefix, stripeId), nil)
	if err != nil {
		return false, fmt.Errorf("failed to check payment method: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return false, fmt.Errorf("failed to check payment method with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ Exists bool }](r)
	return resp.Exists, err
}

func (c *BillingClient) CreateStripeCustomer(name string) (string, error) {
	prefix := "/v1"
	type Body struct{ Name string }
	body, err := marshal(Body{name})
	if err != nil {
		return "", fmt.Errorf("could not serialize request body: %w", err)
	}

	r, err := c.client.Post(prefix+"/organizations/stripe-customer", body)
	if err != nil {
		return "", fmt.Errorf("failed to create stripe customer: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return "", fmt.Errorf("failed to create stripe customer with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[struct{ StripeCustomerId string }](r)
	return resp.StripeCustomerId, err
}
