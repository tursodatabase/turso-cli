package clients

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	base  *url.URL
	token string
}

func New(base *url.URL, token string) *Client {
	return &Client{base, token}
}

func (t *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := t.base.JoinPath(path).String()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", t.token))
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

func (t *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := t.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func (t *Client) Get(path string, body io.Reader) (*http.Response, error) {
	return t.do("GET", path, body)
}

func (t *Client) Post(path string, body io.Reader) (*http.Response, error) {
	return t.do("POST", path, body)
}

func (t *Client) Delete(path string, body io.Reader) (*http.Response, error) {
	return t.do("DELETE", path, body)
}
