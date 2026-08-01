package cmd

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type authRoundTripFunc func(*http.Request) (*http.Response, error)

func (f authRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRelaySessionLifecycle(t *testing.T) {
	original := relayHTTPClient
	t.Cleanup(func() { relayHTTPClient = original })
	requests := []string{}
	relayHTTPClient = &http.Client{Transport: authRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer relay-secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		status, body := http.StatusNoContent, ""
		switch r.Method {
		case http.MethodPost:
			status, body = http.StatusCreated, `{"status":"pending"}`
		case http.MethodGet:
			status, body = http.StatusOK, `{"status":"ready","code":"etsy-code"}`
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	state := strings.Repeat("a", 43)
	if err := createRelaySession(context.Background(), "https://relay.test/", "relay-secret", state); err != nil {
		t.Fatal(err)
	}
	code, err := pollRelaySession(context.Background(), "https://relay.test", "relay-secret", state)
	if err != nil {
		t.Fatal(err)
	}
	if code != "etsy-code" {
		t.Fatalf("code = %q", code)
	}
	if err := deleteRelaySession(context.Background(), "https://relay.test", "relay-secret", state); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /oauth/cli/sessions", "GET /oauth/cli/sessions/" + state, "DELETE /oauth/cli/sessions/" + state}
	if strings.Join(requests, "|") != strings.Join(want, "|") {
		t.Fatalf("requests = %v", requests)
	}
}
