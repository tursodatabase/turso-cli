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
	type ErrorResponse struct {
		Error interface{} `json:"error"`
		Code  string      `json:"code"`
	}
	if result, err := unmarshal[ErrorResponse](res); err == nil {
		if result.Code == "feature_not_available_for_starter_plan" {
			return fmt.Errorf("%s", result.Error)
		}
		if result.Error != nil {
			return fmt.Errorf("%v", result.Error)
		}
	}
	return fmt.Errorf("response failed with status %s", res.Status)
}
