package turso

import (
	"fmt"
	"net/http"
)

type ApiToken struct {
	ID     string `json:"dbId"`
	Name   string
	Owner  uint
	PubKey []byte
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

type CreateApiTokenResponse struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Token string `json:"token"`
}

func (a *ApiTokensClient) Create(name string) (CreateApiTokenResponse, error) {
	url := fmt.Sprintf("/v1/auth/api-tokens/%s", name)

	res, err := a.client.Post(url, nil)
	if err != nil {
		return CreateApiTokenResponse{}, fmt.Errorf("failed to create token: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return CreateApiTokenResponse{}, parseResponseError(res)
	}

	data, err := unmarshal[CreateApiTokenResponse](res)
	if err != nil {
		return CreateApiTokenResponse{}, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return data, nil
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
