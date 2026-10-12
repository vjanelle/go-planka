package planka

import (
	"context"
	"fmt"
	"net/http"
)

// AuthClient creates a client with Bearer token authentication.
func AuthClient(url, token string) (*ClientWithResponses, error) {
	return NewClientWithResponses(url+"/api",
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}),
	)
}

// APIKeyClient creates a client with API key authentication.
func APIKeyClient(url, apiKey string) (*ClientWithResponses, error) {
	return NewClientWithResponses(url+"/api",
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("X-Api-Key", apiKey)
			return nil
		}),
	)
}

// Login authenticates with email/username and password, returning an access token.
func Login(ctx context.Context, url, emailOrUsername, password string) (string, error) {
	c, err := NewClientWithResponses(url + "/api")
	if err != nil {
		return "", err
	}
	resp, err := c.CreateAccessTokenWithResponse(ctx, CreateAccessTokenJSONRequestBody{
		EmailOrUsername: emailOrUsername,
		Password:        password,
	})
	if err != nil {
		return "", err
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("login failed: %s", string(resp.Body))
	}
	return resp.JSON200.Item, nil
}
