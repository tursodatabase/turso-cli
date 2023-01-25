package turso

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	client *http.Client

	BaseURL     *url.URL
	AccessToken string

	common    service
	Databases *DatabasesService
}

type service struct {
	client *Client
}

// NewClient returns a new Turso API client.
func NewClient(baseURL, accessToken string, httpClient *http.Client) (*Client, error) {
	url, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	c := &Client{client: httpClient, BaseURL: url, AccessToken: accessToken}
	c.common.client = c
	c.Databases = (*DatabasesService)(&c.common)
	return c, nil
}

// NewRequest creates an Turso API request.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	url, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	var buf io.ReadWriter
	if body != nil {
		switch b := body.(type) {
		case string:
			buf = bytes.NewBufferString(b)
		case []byte:
			buf = bytes.NewBuffer(b)
		}
	}
	req, err := http.NewRequest(method, url.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

// Do sends a Turso API request to the server and waits for a response.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}
