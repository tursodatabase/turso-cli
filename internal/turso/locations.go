package turso

import (
	"fmt"
	"net/http"
	"regexp"
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

func (c *LocationsClient) Closest(regionUrl string) (string, error) {
	r, err := c.client.Get(regionUrl, nil)
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

func LocationsTable(columns []interface{}) table.Table {
	regex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return table.New(columns...).WithWidthFunc(func(s string) int {
		plainText := regex.ReplaceAllString(s, "")
		return utf8.RuneCountInString(plainText)
	})
}
