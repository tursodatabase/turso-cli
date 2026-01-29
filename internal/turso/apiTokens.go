package turso

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ApiToken struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	CreatedAt    string `json:"created_at"`
	Owner        uint   `json:"-"`
	PubKey       []byte `json:"-"`
}

type ApiTokensClient client

func (a *ApiTokensClient) List() ([]ApiToken, error) {
	res, err := a.client.Get("/v1/auth/api-tokens", nil)
	if err != nil {
		return []ApiToken{}, fmt.Errorf("failed to get api tokens list: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get api tokens list: %s", res.Status)
	}

	type ListResponse struct {
		ApiTokens []ApiToken `json:"tokens"`
	}
	resp, err := unmarshal[ListResponse](res)
	return resp.ApiTokens, err
}

type CreateApiToken struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Value string `json:"value"`
}

func (a *ApiTokensClient) Create(name string) (CreateApiToken, error) {
	return a.CreateWithOrg(name, "")
}

func (a *ApiTokensClient) CreateWithOrg(name string, organization string) (CreateApiToken, error) {
	url := fmt.Sprintf("/v2/auth/api-tokens/%s", name)

	var res *http.Response
	var err error

	if organization != "" {
		reqBody := struct {
			Organization string `json:"organization"`
		}{
			Organization: organization,
		}
		jsonData, marshalErr := json.Marshal(reqBody)
		if marshalErr != nil {
			return CreateApiToken{}, fmt.Errorf("failed to marshal request body: %w", marshalErr)
		}
		body := bytes.NewReader(jsonData)
		res, err = a.client.Post(url, body)
	} else {
		res, err = a.client.Post(url, nil)
	}

	if err != nil {
		return CreateApiToken{}, fmt.Errorf("failed to create token: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return CreateApiToken{}, parseResponseError(res)
	}

	type CreateApiTokenResponse struct {
		ApiToken CreateApiToken `json:"token"`
	}

	data, err := unmarshal[CreateApiTokenResponse](res)
	if err != nil {
		return CreateApiToken{}, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return data.ApiToken, nil
}

func (a *ApiTokensClient) Revoke(name string) error {
	url := fmt.Sprintf("/v1/auth/api-tokens/%s", name)

	res, err := a.client.Delete(url, nil)
	if err != nil {
		return fmt.Errorf("failed to revoke API token: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}

	return nil
}
