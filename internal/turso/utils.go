package turso

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func unmarshal[T any](r *http.Response) (T, error) {
	d, err := io.ReadAll(r.Body)
	t := new(T)
	if err != nil {
		return *t, err
	}
	err = json.Unmarshal(d, &t)
	return *t, err
}

func marshal(data interface{}) (io.Reader, error) {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(data)
	return buf, err
}

func parseResponseError(res *http.Response) error {
	type ErrorResponse struct{ Error interface{} }
	if result, err := unmarshal[ErrorResponse](res); err == nil {
		return fmt.Errorf("%s", result.Error)
	}
	return fmt.Errorf("response failed with status %s", res.Status)
}

type Regions struct {
	Ids             []string `json:"regionIds"`
	Descriptions    []string `json:"regionDescriptions"`
	DefaultRegionId string   `json:"defaultRegionId"`
}

func GetRegions(client *Client) (Regions, error) {
	r, err := client.Get("/v2/regions", nil)
	if err != nil {
		return Regions{}, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return Regions{}, fmt.Errorf("unable to fetch regions: %s", r.Status)
	}

	resp, err := unmarshal[Regions](r)
	if err != nil {
		return Regions{}, err
	}

	return resp, nil
}
