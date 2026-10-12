// Copyright (c) Ippon
// SPDX-License-Identifier: MPL-2.0

package admin_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/ippontech/terraform-provider-anthropic/internal/admin"
)

func newSDKAdminClient(t *testing.T, srv *httptest.Server) *admin.Client {
	t.Helper()
	sdk := anthropic.NewClient(
		option.WithoutEnvironmentDefaults(),
		option.WithBaseURL(srv.URL),
		option.WithAuthToken("bearer-token"),
	)
	c := admin.NewSDKClient(&sdk)
	c.MaxRetries = 0
	return c
}

func TestSDKClientSendsTheBearerAndNoApiKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer bearer-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("x-api-key"); got != "" {
			t.Errorf("x-api-key = %q, want none", got)
		}
		if r.URL.Path != "/v1/organizations/api_keys" || r.URL.Query().Get("limit") != "1" {
			t.Errorf("request URI = %q", r.URL.RequestURI())
		}
		_, _ = w.Write([]byte(`{"data":[],"has_more":false}`))
	}))
	defer srv.Close()

	body, err := newSDKAdminClient(t, srv).DoRequest(context.Background(), http.MethodGet, "/v1/organizations/api_keys?limit=1", nil)
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	if string(body) != `{"data":[],"has_more":false}` {
		t.Errorf("body = %s", body)
	}
}

func TestSDKClientSendsTheRequestBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		var got map[string]string
		if err := json.Unmarshal(raw, &got); err != nil || got["name"] != "ws" {
			t.Errorf("body = %s", raw)
		}
		_, _ = w.Write([]byte(`{"id":"wrkspc_1"}`))
	}))
	defer srv.Close()

	if _, err := newSDKAdminClient(t, srv).DoRequest(context.Background(), http.MethodPost, "/v1/organizations/workspaces", map[string]string{"name": "ws"}); err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
}

func TestSDKClientReportsErrorStatusesAsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"not_found_error","message":"no such workspace"}}`))
	}))
	defer srv.Close()

	_, err := newSDKAdminClient(t, srv).DoRequest(context.Background(), http.MethodGet, "/v1/organizations/workspaces/wrkspc_x", nil)
	if !admin.IsNotFound(err) {
		t.Fatalf("err = %v, want a 404 APIError", err)
	}
}
