package turso

import (
	"fmt"
	"net/http"
)

type UsersClient client

type UserInfo struct {
	Username string `json:"username"`
	Plan     string `json:"plan"`
}

type UserInfoResponse struct {
	User UserInfo `json:"user"`
}

func (u *UsersClient) GetUser() (UserInfo, error) {
	res, err := u.client.Get("/v1/current-user", nil)
	if err != nil {
		return UserInfo{}, fmt.Errorf("failed to get user info: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return UserInfo{}, parseResponseError(res)
	}

	data, err := unmarshal[UserInfoResponse](res)
	if err != nil {
		return UserInfo{}, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return data.User, nil
}
