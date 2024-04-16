package turso

import (
	"fmt"
	"net/http"
)

type InvoicesClient client

type Invoice struct {
	Number          string `json:"invoice_number"`
	Amount          string `json:"amount_due"`
	DueDate         string `json:"due_date"`
	PaidAt          string `json:"paid_at"`
	PaymentFailedAt string `json:"payment_failed_at"`
	InvoicePdf      string `json:"invoice_pdf"`
}

func (i *InvoicesClient) List() ([]Invoice, error) {
	r, err := i.client.Get(i.URL(""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoices: %w", err)
	}
	defer r.Body.Close()

	org := i.client.Org
	if isNotMemberErr(r.StatusCode, org) {
		return nil, notMemberErr(org)
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get invoices: received status code%w", parseResponseError(r))
	}

	type ListResponse struct {
		Invoices []Invoice `json:"invoices"`
	}
	resp, err := unmarshal[ListResponse](r)
	return resp.Invoices, err
}

func (i *InvoicesClient) URL(suffix string) string {
	prefix := "/v1"
	if i.client.Org != "" {
		prefix = "/v1/organizations/" + i.client.Org
	}
	return prefix + "/invoices" + suffix
}
