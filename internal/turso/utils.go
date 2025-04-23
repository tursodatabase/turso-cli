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

type ErrorResponseDetails struct {
	Error interface{} `json:"error"`
	Code  string      `json:"code"`
}

func GetFeatureError(body []byte, feature string) error {
	if len(body) == 0 {
		return nil
	}

	var errResp ErrorResponseDetails
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("failed to unmarshal error response: %w", err)
	}

	if errResp.Code == "feature_not_available_for_starter_plan" {
		return fmt.Errorf("%s are not available on the free plan - upgrade to access this feature", feature)
	}
	return nil
}

func parseResponseError(res *http.Response) error {
	d, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("response failed with status %s", res.Status)
	}

	var errResp ErrorResponseDetails
	if err := json.Unmarshal(d, &errResp); err == nil {
		if errResp.Error != nil {
			return fmt.Errorf("%v", errResp.Error)
		}
	}
	return fmt.Errorf("response failed with status %s", res.Status)
}
