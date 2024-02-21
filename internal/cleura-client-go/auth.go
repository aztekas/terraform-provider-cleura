package cleura

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetToken - Get a new token for user
func (c *Client) GetToken() (*AuthResponse, error) {
	if c.Auth.Username == "" || c.Auth.Password == "" {
		return nil, fmt.Errorf("define username and password")
	}
	type AuthStructWrapper struct {
		Auth AuthStruct `json:"auth"`
	}
	auth := AuthStructWrapper{
		Auth: c.Auth,
	}
	rb, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/v1/tokens", c.HostURL), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req, 200)
	if err != nil {
		return nil, err
	}

	ar := AuthResponse{}
	err = json.Unmarshal(body, &ar)
	if err != nil {
		return nil, err
	}
	return &ar, nil
}

// RevokeToken - Revoke client token
func (c *Client) RevokeToken() error {
	//https://rest.cleura.cloud/auth/v1/tokens
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/auth/v1/tokens", c.HostURL), nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, 204)
	if err != nil {
		return err
	}
	return nil
}

// Validate client token
func (c *Client) ValidateToken() error {
	//https://rest.cleura.cloud/auth/v1/tokens/validate

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/v1/tokens/validate", c.HostURL), nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, 204)
	if err != nil {
		return err
	}
	return nil
}
