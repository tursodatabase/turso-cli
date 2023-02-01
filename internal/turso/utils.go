package turso

import (
	"bytes"
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

func Marshal(data interface{}) (io.Reader, error) {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(data)
	return buf, err
}
