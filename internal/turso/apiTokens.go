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
