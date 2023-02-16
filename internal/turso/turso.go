package turso

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

// Collection of all turso clients
type Client struct {
	baseUrl *url.URL
	token   *string

	// Single instance to be reused by all clients
	base *client

	Instances *InstancesClient
	Databases *DatabasesClient
}

// Client struct that will be aliases by all other clients
type client struct {
	client *Client
}

func New(base *url.URL, token *string) *Client {
	c := &Client{baseUrl: base, token: token}

	c.base = &client{c}
	c.Instances = (*InstancesClient)(c.base)
	c.Databases = (*DatabasesClient)(c.base)
	return c
}

func (t *Client) newRequest(method, urlPath string, body io.Reader) (*http.Request, error) {
	url, err := url.Parse(t.baseUrl.String())
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, urlPath)
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, err
	}
	if t.token != nil {
		req.Header.Add("Authorization", fmt.Sprint("Bearer ", *t.token))
	}
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

func (t *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := t.newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%s - please login with %s", resp.Status, Emph("turso auth login"))
	}
	return resp, nil
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
