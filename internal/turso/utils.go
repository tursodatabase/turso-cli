package turso

import (
	"encoding/json"
	"io"
	"net/http"
)

func Unmarshall[T any](r *http.Response) (T, error) {
	d, err := io.ReadAll(r.Body)
	t := new(T)
	if err != nil {
		return *t, err
	}
	json.Unmarshal(d, &t)
	return *t, nil
}
