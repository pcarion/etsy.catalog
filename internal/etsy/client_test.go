package etsy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testClient(fn roundTripFunc) *Client {
	c := New("key:secret", "token")
	c.BaseURL = "https://example.test"
	c.HTTP = &http.Client{Transport: fn}
	return c
}

func TestDoAddsAuthenticationAndDecodes(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("x-api-key") != "key:secret" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"listing_id":42}`)), Header: make(http.Header)}, nil
	})
	var got map[string]any
	if err := c.Do(context.Background(), http.MethodGet, "listings/42", nil, nil, &got); err != nil {
		t.Fatal(err)
	}
	if got["listing_id"] != float64(42) {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestDoReturnsAPIError(t *testing.T) {
	c := testClient(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"error":"bad"}`)), Header: make(http.Header)}, nil
	})
	err := c.Do(context.Background(), http.MethodGet, "bad", nil, nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Status != 400 {
		t.Fatalf("got %T %v", err, err)
	}
}
