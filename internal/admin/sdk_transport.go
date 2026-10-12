// Copyright (c) Ippon
// SPDX-License-Identifier: MPL-2.0

package admin

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// NewSDKClient returns an Admin API client that authenticates through sdk
// rather than with an Admin API key. An org:admin OAuth bearer reaches the
// whole Admin API, so a provider holding only that bearer — from auth_token or
// a federation block — can still manage workspaces, members and API keys.
// The SDK client keeps the bearer fresh; this client keeps its own retries.
func NewSDKClient(sdk *anthropic.Client) *Client {
	return &Client{
		BaseURL:    adminAPIBaseURL,
		HTTPClient: &http.Client{Timeout: 60 * time.Second, Transport: sdkTransport{sdk: sdk}},
		MaxRetries: DefaultMaxRetries,
	}
}

// sdkTransport sends each request through an SDK client, so it carries the
// credential that client holds. The SDK's own retries are off: Client.DoRequest
// decides what to replay, and must see every response to do so.
type sdkTransport struct {
	sdk *anthropic.Client
}

func (t sdkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	opts := []option.RequestOption{option.WithMaxRetries(0)}
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		opts = append(opts, option.WithRequestBody(req.Header.Get("content-type"), body))
	}

	var resp *http.Response
	err := t.sdk.Execute(req.Context(), req.Method, strings.TrimPrefix(req.URL.RequestURI(), "/"), nil, &resp, opts...)
	// The SDK hands back the response for an error status too, with the body
	// restored; Client.attempt turns that into an APIError itself.
	if resp != nil {
		return resp, nil
	}
	return nil, err
}
