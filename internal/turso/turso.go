package turso

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/tursodatabase/turso-cli/internal/flags"
)

// Collection of all turso clients
type Client struct {
	baseUrl    *url.URL
	token      string
	cliVersion string
	Org        string

	// Single instance to be reused by all clients
	base *client

	Instances     *InstancesClient
	Databases     *DatabasesClient
	Organizations *OrganizationsClient
	ApiTokens     *ApiTokensClient
	Locations     *LocationsClient
	Tokens        *TokensClient
	Users         *UsersClient
	Plans         *PlansClient
	Subscriptions *SubscriptionClient
	Billing       *BillingClient
	Groups        *GroupsClient
	Invoices      *InvoicesClient
}

// Client struct that will be aliases by all other clients
type client struct {
	client *Client
}

func New(base *url.URL, token string, cliVersion string, org string) *Client {
	c := &Client{baseUrl: base, token: token, cliVersion: cliVersion, Org: org}

	c.base = &client{c}
	c.Instances = (*InstancesClient)(c.base)
	c.Databases = (*DatabasesClient)(c.base)
	c.Organizations = (*OrganizationsClient)(c.base)
	c.ApiTokens = (*ApiTokensClient)(c.base)
	c.Locations = (*LocationsClient)(c.base)
	c.Tokens = (*TokensClient)(c.base)
	c.Users = (*UsersClient)(c.base)
	c.Plans = (*PlansClient)(c.base)
	c.Subscriptions = (*SubscriptionClient)(c.base)
	c.Billing = (*BillingClient)(c.base)
	c.Groups = (*GroupsClient)(c.base)
	c.Invoices = (*InvoicesClient)(c.base)
	return c
}

func (t *Client) newRequest(method, urlPath string, body io.Reader, extraHeaders map[string]string) (*http.Request, error) {
	if _, exists := extraHeaders["Content-Type"]; !exists {
		return nil, fmt.Errorf("content type is required")
	}
	url, err := url.Parse(t.baseUrl.String())
	if err != nil {
		return nil, err
	}
	url, err = url.Parse(urlPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, err
	}
	if t.token != "" {
		req.Header.Add("Authorization", fmt.Sprint("Bearer ", t.token))
	}
	req.Header.Add("TursoCliVersion", t.cliVersion)
	parsedCliVersion := t.cliVersion
	if parsedCliVersion != "dev" {
		parsedCliVersion = t.cliVersion[1:]
	}
	req.Header.Add("User-Agent", fmt.Sprintf("turso-cli/%s (%s/%s)", parsedCliVersion, runtime.GOOS, runtime.GOARCH))
	for header, value := range extraHeaders {
		req.Header.Add(header, value)
	}
	return req, nil
}

func (t *Client) do(method, path string, body io.Reader, extraHeaders map[string]string) (*http.Response, error) {
	req, err := t.newRequest(method, path, body, extraHeaders)
	var reqDump string
	if flags.Debug() {
		reqDump = dumpRequest(req)
	}
	if err != nil {
		return nil, err
	}

	var client = &http.Client{
		// go seems to have issues resolving things like region.localhost:9090, while curl works fine.
		// Make sure we can get it to work in those cases by resolving it manually.
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}

				if strings.HasSuffix(host, ".localhost") {
					return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort("127.0.0.1", port))
				}

				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if flags.Debug() {
		printDumps(reqDump, dumpResponse(resp))
	}
	return resp, nil
}

func printDumps(req, resp string) {
	if req != "" {
		fmt.Println(req)
	}
	if resp != "" {
		fmt.Println(resp)
	}
}

func dumpRequest(req *http.Request) string {
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return ""
	}
	return string(dump)
}

func dumpResponse(req *http.Response) string {
	dump, err := httputil.DumpResponse(req, true)
	if err != nil {
		return ""
	}
	return string(dump)
}

func Header(key, value string) map[string]string {
	return map[string]string{
		key: value,
	}
}

func (t *Client) Get(path string, body io.Reader) (*http.Response, error) {
	return t.do("GET", path, body, Header("Content-Type", "application/json"))
}

func (t *Client) GetWithHeaders(path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	headers["Content-Type"] = "application/json"
	return t.do("GET", path, body, headers)
}

func (t *Client) Post(path string, body io.Reader) (*http.Response, error) {
	return t.do("POST", path, body, Header("Content-Type", "application/json"))
}

func (t *Client) PostBinary(path string, body io.Reader) (*http.Response, error) {
	return t.do("POST", path, body, Header("Content-Type", "application/octet-stream"))
}

func (t *Client) Patch(path string, body io.Reader) (*http.Response, error) {
	return t.do("PATCH", path, body, Header("Content-Type", "application/json"))
}

func (t *Client) Put(path string, body io.Reader) (*http.Response, error) {
	return t.do("PUT", path, body, Header("Content-Type", "application/json"))
}

func (t *Client) Upload(path string, fileData *os.File) (*http.Response, error) {
	body, bodyWriter := io.Pipe()
	writer := multipart.NewWriter(bodyWriter)
	go func() {
		formFile, err := writer.CreateFormFile("file", fileData.Name())
		if err != nil {
			bodyWriter.CloseWithError(err)
			return
		}
		if _, err := io.Copy(formFile, fileData); err != nil {
			bodyWriter.CloseWithError(err)
			return
		}
		bodyWriter.CloseWithError(writer.Close())
	}()
	req, err := t.newRequest("POST", path, body, Header("Content-Type", writer.FormDataContentType()))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *Client) Delete(path string, body io.Reader) (*http.Response, error) {
	return t.do("DELETE", path, body, Header("Content-Type", "application/json"))
}
