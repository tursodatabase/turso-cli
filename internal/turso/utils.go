package turso

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"
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

func CheckName(name string) error {
	if len(name) == 0 || len(name) > 32 {
		return fmt.Errorf("name must be between 1 and 32 characters long")
	}

	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("name cannot start or end with a hyphen")
	}

	for _, r := range name {
		if !(unicode.IsDigit(r) || (unicode.IsLetter(r) && unicode.IsLower(r)) || r == '-') {
			return fmt.Errorf("name can only contain lowercase letters, numbers, and hyphens")
		}
	}
	return nil
}
