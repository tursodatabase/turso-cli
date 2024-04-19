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

type BillingAddress struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type TaxID struct {
	Value   string `json:"value"`
	Country string `json:"country"`
}
type BillingCustomer struct {
	Name           string         `json:"name"`
	Email          string         `json:"email"`
	TaxID          TaxID          `json:"tax_id"`
	BillingAddress BillingAddress `json:"billing_address"`
}

func (c *BillingClient) GetBillingCustomer() (BillingCustomer, error) {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}
	r, err := c.client.Get(prefix+"/billing/customer", nil)
	if err != nil {
		return BillingCustomer{}, fmt.Errorf("failed to get billing customer: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return BillingCustomer{}, fmt.Errorf("failed to get billing customer with status %s: %v", r.Status, parseResponseError(r))
	}

	resp, err := unmarshal[BillingCustomer](r)
	return resp, err
}

func (c *BillingClient) UpdateBillingCustomer(customer BillingCustomer) error {
	prefix := "/v1"
	if c.client.Org != "" {
		prefix = "/v1/organizations/" + c.client.Org
	}
	body, err := marshal(customer)
	if err != nil {
		return fmt.Errorf("could not serialize request body: %w", err)
	}
	r, err := c.client.Put(prefix+"/billing/customer", body)
	if err != nil {
		return fmt.Errorf("failed to update billing customer: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return fmt.Errorf("failed to update billing customer with status %s: %v", r.Status, parseResponseError(r))
	}

	return nil
}
