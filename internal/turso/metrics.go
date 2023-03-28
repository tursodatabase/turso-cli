package turso

import (
	"fmt"
	"net/http"
)

type Usage struct {
	Total int `json:"total"`
}

type Metrics struct {
	Usage Usage `json:"usage"`
}

type MetricsClient client

func (i *MetricsClient) Show() (*Metrics, error) {
	r, err := i.client.Get("v1/metrics/volumes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response with status code %d", r.StatusCode)
	}
	resp, err := unmarshal[Metrics](r)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
