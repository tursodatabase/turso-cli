package turso

import (
	"fmt"
	"net/http"
)

type TokensClient client

func (c *TokensClient) Validate(token string) (int64, error) {
	r, err := c.client.Get("/v1/auth/validate", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to request validation: %s", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to validate token: %s", r.Status)
	}

	data, err := unmarshal[struct{ Exp int64 }](r)
	if err != nil {
		return 0, fmt.Errorf("failed to deserialize validate token response: %w", err)
	}

	return data.Exp, nil
}
