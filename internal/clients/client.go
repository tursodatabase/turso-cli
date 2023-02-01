package clients

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type client struct {
	base  *url.URL
	token string
}

func (t *client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := t.base.JoinPath(path).String()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", t.token))
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

func (t *client) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := t.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func (t *client) Get(path string, body io.Reader) (*http.Response, error) {
	return t.do("GET", path, body)
}

func (t *client) Post(path string, body io.Reader) (*http.Response, error) {
	return t.do("POST", path, body)
}

func (t *client) Delete(path string, body io.Reader) (*http.Response, error) {
	return t.do("DELETE", path, body)
}
