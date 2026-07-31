package etsy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL, APIKey, Token string
	HTTP                   *http.Client
}

func New(apiKey, token string) *Client {
	return &Client{BaseURL: "https://api.etsy.com/v3/application", APIKey: apiKey, Token: token, HTTP: &http.Client{Timeout: 45 * time.Second}}
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Etsy API returned HTTP %d: %s", e.Status, e.Body)
}

func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	u := strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, r)
	if err != nil {
		return err
	}
	c.headers(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("call Etsy API: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("decode Etsy response: %w", err)
		}
	}
	return nil
}

func (c *Client) DoForm(ctx context.Context, method, path string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+"/"+strings.TrimLeft(path, "/"), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	c.headers(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("call Etsy API: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("decode Etsy response: %w", err)
		}
	}
	return nil
}

func (c *Client) Upload(ctx context.Context, path, field, filename string, fields map[string]string, out any) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return err
		}
	}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	p, err := w.CreateFormFile(field, filepath.Base(filename))
	if err != nil {
		return err
	}
	if _, err = io.Copy(p, f); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/"+strings.TrimLeft(path, "/"), &body)
	if err != nil {
		return err
	}
	c.headers(req)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}
	if out != nil && len(b) > 0 {
		return json.Unmarshal(b, out)
	}
	return nil
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("Authorization", "Bearer "+c.Token)
}
