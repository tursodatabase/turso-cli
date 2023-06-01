package turso

import (
	"fmt"
	"net/http"
	"time"
)

type LocationsClient client

type LocationsResponse struct {
	Locations map[string]string
}

func (c *LocationsClient) Get() (map[string]string, error) {
	r, err := c.client.Get("/v1/locations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request locations: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get locations: %s", r.Status)

	}

	data, err := unmarshal[LocationsResponse](r)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize locations response: %w", err)
	}

	return data.Locations, nil
}

type ClosestLocationResponse struct {
	Server string
}

func (c *LocationsClient) Closest() (string, error) {
	r, err := c.client.Get("https://region.turso.io", nil)
	if err != nil {
		return "", fmt.Errorf("failed to request closest: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get closest location: %s", r.Status)

	}

	data, err := unmarshal[ClosestLocationResponse](r)
	if err != nil {
		return "", fmt.Errorf("failed to deserialize locations response: %w", err)
	}

	return data.Server, nil
}

func ProbeLocation(location string) *time.Duration {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://region.turso.io:8080/", nil)
	if err != nil {
		return nil
	}
	req.Header.Add("fly-prefer-region", location)

	start := time.Now()
	_, err = client.Do(req)
	if err != nil {
		return nil
	}
	dur := time.Since(start)
	return &dur
}
