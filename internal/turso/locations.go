package turso

import (
	"fmt"
	"net/http"
)

type LocationsClient client

type LocationsResponse struct {
	Locations map[string]string
	Default   string
}

func (c *LocationsClient) Get() (LocationsResponse, error) {
	r, err := c.client.Get("/v1/locations", nil)
	if err != nil {
		return LocationsResponse{}, fmt.Errorf("failed to request locations: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return LocationsResponse{}, fmt.Errorf("failed to get locations: %s", r.Status)

	}

	data, err := unmarshal[LocationsResponse](r)
	if err != nil {
		return LocationsResponse{}, fmt.Errorf("failed to deserialize locations response: %w", err)
	}

	return data, nil
}
