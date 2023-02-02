package turso

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Collection of all turso clients
type Client struct {
	baseUrl *url.URL
	token   string

	// Single instance to be reused by all clients
	base *client

	Instances *InstancesClient
	Databases *DatabasesClient
}

// Client struct that will be aliases by all other clients
type client struct {
	client *Client
}

func New(base *url.URL, token string) *Client {
	c := &Client{baseUrl: base, token: token}

	c.base = &client{c}
	c.Instances = (*InstancesClient)(c.base)
	c.Databases = (*DatabasesClient)(c.base)
	return c
}

func (t *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := t.baseUrl.JoinPath(path).String()
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
