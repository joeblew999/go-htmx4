// Package cfapi is the small Cloudflare REST client shared by kit/cfdeploy and kit/cftail: bearer auth
// and the {success, errors, result} envelope.
package cfapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DefaultBaseURL is the Cloudflare API.
const DefaultBaseURL = "https://api.cloudflare.com/client/v4"

// Client calls the API for one account.
type Client struct {
	Token     string
	AccountID string
	BaseURL   string       // default DefaultBaseURL
	HTTP      *http.Client // default http.DefaultClient
}

// Body is a request body with its content type.
type Body struct {
	ContentType string
	Data        []byte
}

// JSON encodes v as a request body.
func JSON(v any) *Body {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return &Body{ContentType: "application/json", Data: b}
}

// AccountPath prefixes p with /accounts/{AccountID}.
func (c *Client) AccountPath(p string) string { return "/accounts/" + c.AccountID + p }

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Do calls method apiPath and decodes the envelope's result into out (if non-nil). bearer overrides the
// API token (asset uploads authenticate with an upload-session JWT).
func (c *Client) Do(ctx context.Context, method, apiPath, bearer string, body *Body, out any) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body.Data)
	}
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, method, base+apiPath, r)
	if err != nil {
		return err
	}
	if bearer == "" {
		bearer = c.Token
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	if body != nil {
		req.Header.Set("Content-Type", body.ContentType)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env struct {
		Success bool            `json:"success"`
		Errors  []apiError      `json:"errors"`
		Result  json.RawMessage `json:"result"`
	}
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s %s: HTTP %d, non-JSON response", method, apiPath, resp.StatusCode)
	}
	if !env.Success {
		return fmt.Errorf("%s %s: HTTP %d: %v", method, apiPath, resp.StatusCode, env.Errors)
	}
	if out != nil && len(env.Result) > 0 && string(env.Result) != "null" {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}
