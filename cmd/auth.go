package cmd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	authorizeURL = "https://www.etsy.com/oauth/connect"
	tokenURL     = "https://api.etsy.com/v3/public/oauth/token"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func newAuthCmd() *cobra.Command {
	c := &cobra.Command{Use: "auth", Short: "Authorize with Etsy and refresh OAuth tokens"}
	c.AddCommand(newAuthURLCmd(), newAuthExchangeCmd(), newAuthRefreshCmd())
	return c
}

func newAuthURLCmd() *cobra.Command {
	var redirectURI, scopes string
	c := &cobra.Command{Use: "url", Short: "Generate an Etsy PKCE authorization URL", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error {
		key, err := keystring()
		if err != nil {
			return err
		}
		if redirectURI == "" {
			return fmt.Errorf("--redirect-uri is required")
		}
		verifier, err := randomURLSafe(64)
		if err != nil {
			return fmt.Errorf("generate PKCE verifier: %w", err)
		}
		state, err := randomURLSafe(32)
		if err != nil {
			return fmt.Errorf("generate OAuth state: %w", err)
		}
		sum := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(sum[:])
		q := url.Values{"response_type": {"code"}, "client_id": {key}, "redirect_uri": {redirectURI}, "scope": {strings.Join(strings.Fields(scopes), " ")}, "state": {state}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
		fmt.Fprintf(opts.out, "Open this URL in a browser:\n\n%s?%s\n\nSave these values until the exchange step:\nETSY_CODE_VERIFIER=%s\nETSY_OAUTH_STATE=%s\n", authorizeURL, q.Encode(), verifier, state)
		return nil
	}}
	c.Flags().StringVar(&redirectURI, "redirect-uri", "", "HTTPS redirect URI registered for the Etsy app")
	c.Flags().StringVar(&scopes, "scopes", "listings_r listings_w", "space-separated OAuth scopes")
	return c
}

func newAuthExchangeCmd() *cobra.Command {
	var code, verifier, redirectURI string
	c := &cobra.Command{Use: "exchange", Short: "Exchange an authorization code for tokens", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		key, err := keystring()
		if err != nil {
			return err
		}
		if code == "" || verifier == "" || redirectURI == "" {
			return fmt.Errorf("--code, --verifier, and --redirect-uri are required")
		}
		return requestToken(cmd.Context(), url.Values{"grant_type": {"authorization_code"}, "client_id": {key}, "redirect_uri": {redirectURI}, "code": {code}, "code_verifier": {verifier}})
	}}
	c.Flags().StringVar(&code, "code", "", "authorization code returned by Etsy")
	c.Flags().StringVar(&verifier, "verifier", "", "PKCE verifier printed by auth url")
	c.Flags().StringVar(&redirectURI, "redirect-uri", "", "same redirect URI used by auth url")
	return c
}

func newAuthRefreshCmd() *cobra.Command {
	var refreshToken string
	c := &cobra.Command{Use: "refresh", Short: "Exchange a refresh token for fresh tokens", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		key, err := keystring()
		if err != nil {
			return err
		}
		if refreshToken == "" {
			return fmt.Errorf("missing refresh token: set ETSY_REFRESH_TOKEN or use --refresh-token")
		}
		return requestToken(cmd.Context(), url.Values{"grant_type": {"refresh_token"}, "client_id": {key}, "refresh_token": {refreshToken}})
	}}
	c.Flags().StringVar(&refreshToken, "refresh-token", os.Getenv("ETSY_REFRESH_TOKEN"), "refresh token (or ETSY_REFRESH_TOKEN)")
	return c
}

func requestToken(ctx context.Context, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h := &http.Client{Timeout: 30 * time.Second}
	resp, err := h.Do(req)
	if err != nil {
		return fmt.Errorf("request Etsy token: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Etsy token endpoint returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var token tokenResponse
	if err := json.Unmarshal(b, &token); err != nil {
		return fmt.Errorf("decode Etsy token response: %w", err)
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		return fmt.Errorf("Etsy token response did not contain access and refresh tokens")
	}
	fmt.Fprintf(opts.out, "ETSY_ACCESS_TOKEN=%s\nETSY_REFRESH_TOKEN=%s\nEXPIRES_IN=%d\n", token.AccessToken, token.RefreshToken, token.ExpiresIn)
	return nil
}

func keystring() (string, error) {
	if opts.apiKey == "" {
		return "", fmt.Errorf("missing API key: set ETSY_API_KEY or use --api-key")
	}
	return strings.SplitN(opts.apiKey, ":", 2)[0], nil
}

func randomURLSafe(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
