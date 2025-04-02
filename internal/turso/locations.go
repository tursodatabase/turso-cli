package turso

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/rodaine/table"
)

type LocationsClient client

type LocationsResponse struct {
	Locations map[string]string
}

type Location struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type LocationResponse struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Closest     []Location
}

func (c *LocationsClient) Get(location string) (LocationResponse, error) {
	r, err := c.client.Get("/v1/locations/"+location, nil)
	if err != nil {
		return LocationResponse{}, fmt.Errorf("failed to request location %s: %w", location, err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return LocationResponse{}, fmt.Errorf("failed to get location %s: %s", location, r.Status)
	}

	data, err := unmarshal[struct {
		Location LocationResponse `json:"location"`
	}](r)

	if err != nil {
		return LocationResponse{}, fmt.Errorf("failed to deserialize location response: %w", err)
	}

	return data.Location, nil
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
		return "", fmt.Errorf("failed to get closest location: %w", parseResponseError(r))

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
	r, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer r.Body.Close()

	dur := time.Since(start)
	if r.StatusCode != http.StatusOK {
		return nil
	}
	data, err := unmarshal[ClosestLocationResponse](r)
	if err != nil {
		return nil
	}
	if data.Server != location {
		return nil
	}
	return &dur
}

func LocationsTable(columns []interface{}) table.Table {
	regex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return table.New(columns...).WithWidthFunc(func(s string) int {
		plainText := regex.ReplaceAllString(s, "")
		return utf8.RuneCountInString(plainText)
	})
}
