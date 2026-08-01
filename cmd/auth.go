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
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/pcarion/etsy.catalog/internal/authstore"
	"github.com/spf13/cobra"
)

const (
	authorizeURL = "https://www.etsy.com/oauth/connect"
	tokenURL     = "https://api.etsy.com/v3/public/oauth/token"
)

var relayHTTPClient = &http.Client{Timeout: 30 * time.Second}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func newAuthCmd() *cobra.Command {
	c := &cobra.Command{Use: "auth", Short: "Authorize with Etsy and refresh OAuth tokens"}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown auth command %q", args[0])
		}
		return cmd.Help()
	}
	c.AddCommand(newAuthLoginCmd(), newAuthURLCmd(), newAuthRefreshCmd())
	return c
}

func newAuthLoginCmd() *cobra.Command {
	var redirectURI, relayURL, callbackToken, scopes string
	var noOpen bool
	c := &cobra.Command{Use: "login", Short: "Complete Etsy authorization using the callback relay", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		key, err := keystring()
		if err != nil {
			return err
		}
		if redirectURI == "" {
			return fmt.Errorf("missing redirect URI: set ETSY_AUTH_REDIRECT_URL or use --redirect-uri")
		}
		if relayURL == "" {
			return fmt.Errorf("missing relay URL: set ETSY_AUTH_RELAY_URL or use --relay-url")
		}
		if callbackToken == "" {
			return fmt.Errorf("missing callback token: set ETSY_AUTH_CALLBACK_TOKEN or use --callback-token")
		}
		if strings.HasSuffix(strings.TrimRight(redirectURI, "/"), "/oauth/callback") {
			return fmt.Errorf("ETSY_AUTH_REDIRECT_URL points to the server OAuth callback; use the CLI callback ending in /oauth/cli/callback")
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
		if err := createRelaySession(cmd.Context(), relayURL, callbackToken, state); err != nil {
			return err
		}
		q := url.Values{"response_type": {"code"}, "client_id": {key}, "redirect_uri": {redirectURI}, "scope": {strings.Join(strings.Fields(scopes), " ")}, "state": {state}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
		authURL := authorizeURL + "?" + q.Encode()
		if noOpen {
			fmt.Fprintf(opts.out, "Open this URL in a browser:\n%s\n", authURL)
		} else if err := openBrowser(authURL); err != nil {
			fmt.Fprintf(opts.err, "Warning: could not open the browser: %v\nOpen this URL manually:\n%s\n", err, authURL)
		}
		fmt.Fprintln(opts.out, "Waiting for Etsy authorization...")
		code, err := pollRelaySession(cmd.Context(), relayURL, callbackToken, state)
		if err != nil {
			return err
		}
		token, err := exchangeToken(cmd.Context(), url.Values{"grant_type": {"authorization_code"}, "client_id": {key}, "redirect_uri": {redirectURI}, "code": {code}, "code_verifier": {verifier}})
		if err != nil {
			return err
		}
		if err := persistToken(token); err != nil {
			return err
		}
		if err := deleteRelaySession(cmd.Context(), relayURL, callbackToken, state); err != nil {
			fmt.Fprintf(opts.err, "Warning: token exchanged but callback session cleanup failed: %v\n", err)
		}
		printToken(token)
		return nil
	}}
	c.Flags().StringVar(&redirectURI, "redirect-uri", os.Getenv("ETSY_AUTH_REDIRECT_URL"), "CLI callback URI (or ETSY_AUTH_REDIRECT_URL)")
	c.Flags().StringVar(&relayURL, "relay-url", os.Getenv("ETSY_AUTH_RELAY_URL"), "callback relay base URL (or ETSY_AUTH_RELAY_URL)")
	c.Flags().StringVar(&callbackToken, "callback-token", os.Getenv("ETSY_AUTH_CALLBACK_TOKEN"), "callback relay bearer token (or ETSY_AUTH_CALLBACK_TOKEN)")
	c.Flags().StringVar(&scopes, "scopes", "listings_r listings_w", "space-separated OAuth scopes")
	c.Flags().BoolVar(&noOpen, "no-open", false, "print the URL without opening a browser")
	return c
}

func newAuthURLCmd() *cobra.Command {
	var redirectURI, scopes string
	var noOpen bool
	c := &cobra.Command{Use: "url", Short: "Generate an Etsy PKCE authorization URL", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error {
		key, err := keystring()
		if err != nil {
			return err
		}
		if redirectURI == "" {
			return fmt.Errorf("missing redirect URI: set ETSY_AUTH_REDIRECT_URL or use --redirect-uri")
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
		authURL := authorizeURL + "?" + q.Encode()
		if !noOpen {
			if err := openBrowser(authURL); err != nil {
				fmt.Fprintf(opts.err, "Warning: could not open the browser: %v\n", err)
			}
		}
		fmt.Fprintf(opts.out, "Authorization URL:\n\n%s\n\nDiagnostic PKCE values (auth url does not complete login):\nETSY_CODE_VERIFIER=%s\nETSY_OAUTH_STATE=%s\n", authURL, verifier, state)
		return nil
	}}
	c.Flags().StringVar(&redirectURI, "redirect-uri", os.Getenv("ETSY_AUTH_REDIRECT_URL"), "HTTPS redirect URI (or ETSY_AUTH_REDIRECT_URL)")
	c.Flags().StringVar(&scopes, "scopes", "listings_r listings_w", "space-separated OAuth scopes")
	c.Flags().BoolVar(&noOpen, "no-open", false, "print the URL without opening a browser")
	return c
}

func openBrowser(target string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command, args = "open", []string{target}
	case "windows":
		command, args = "rundll32", []string{"url.dll,FileProtocolHandler", target}
	default:
		command, args = "xdg-open", []string{target}
	}
	if err := exec.Command(command, args...).Start(); err != nil {
		return fmt.Errorf("run %s: %w", command, err)
	}
	return nil
}

func newAuthRefreshCmd() *cobra.Command {
	var refreshToken string
	stored, _ := authstore.Load()
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
	defaultRefresh := os.Getenv("ETSY_REFRESH_TOKEN")
	if defaultRefresh == "" {
		defaultRefresh = stored.RefreshToken
	}
	c.Flags().StringVar(&refreshToken, "refresh-token", defaultRefresh, "refresh token (or ETSY_REFRESH_TOKEN or saved credentials)")
	return c
}

func requestToken(ctx context.Context, form url.Values) error {
	token, err := exchangeToken(ctx, form)
	if err != nil {
		return err
	}
	if err := persistToken(token); err != nil {
		return err
	}
	printToken(token)
	return nil
}

func persistToken(token tokenResponse) error {
	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	if err := authstore.Save(authstore.Credentials{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: expiresAt}); err != nil {
		return fmt.Errorf("save OAuth credentials: %w", err)
	}
	name, _ := authstore.Path()
	fmt.Fprintf(opts.err, "Saved OAuth credentials to %s\n", name)
	return nil
}

func exchangeToken(ctx context.Context, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h := &http.Client{Timeout: 30 * time.Second}
	resp, err := h.Do(req)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("request Etsy token: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return tokenResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return tokenResponse{}, fmt.Errorf("Etsy token endpoint returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var token tokenResponse
	if err := json.Unmarshal(b, &token); err != nil {
		return tokenResponse{}, fmt.Errorf("decode Etsy token response: %w", err)
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		return tokenResponse{}, fmt.Errorf("Etsy token response did not contain access and refresh tokens")
	}
	return token, nil
}

func printToken(token tokenResponse) {
	fmt.Fprintf(opts.out, "Authentication successful. Access token expires in %d seconds.\n", token.ExpiresIn)
	if os.Getenv("ETSY_ACCESS_TOKEN") != "" || os.Getenv("ETSY_REFRESH_TOKEN") != "" {
		fmt.Fprintln(opts.err, "Warning: exported ETSY_ACCESS_TOKEN or ETSY_REFRESH_TOKEN values override saved credentials; unset them before using the saved login.")
	}
}

func relayRequest(ctx context.Context, method, target, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return relayHTTPClient.Do(req)
}

func createRelaySession(ctx context.Context, base, token, state string) error {
	b, _ := json.Marshal(map[string]string{"state": state})
	target := strings.TrimRight(base, "/") + "/oauth/cli/sessions"
	resp, err := relayRequest(ctx, http.MethodPost, target, token, strings.NewReader(string(b)))
	if err != nil {
		return fmt.Errorf("register callback session: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("callback relay returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

func pollRelaySession(ctx context.Context, base, token, state string) (string, error) {
	target := strings.TrimRight(base, "/") + "/oauth/cli/sessions/" + url.PathEscape(state)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		resp, err := relayRequest(ctx, http.MethodGet, target, token, nil)
		if err != nil {
			return "", fmt.Errorf("poll callback session: %w", err)
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		var result struct{ Status, Code, Error, ErrorDescription string }
		_ = json.Unmarshal(data, &result)
		switch resp.StatusCode {
		case http.StatusOK:
			if result.Code == "" {
				return "", fmt.Errorf("callback relay returned ready without a code")
			}
			return result.Code, nil
		case http.StatusAccepted:
		case http.StatusBadRequest:
			return "", fmt.Errorf("Etsy authorization failed: %s %s", result.Error, result.ErrorDescription)
		default:
			return "", fmt.Errorf("callback relay returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func deleteRelaySession(ctx context.Context, base, token, state string) error {
	target := strings.TrimRight(base, "/") + "/oauth/cli/sessions/" + url.PathEscape(state)
	resp, err := relayRequest(ctx, http.MethodDelete, target, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("callback relay returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
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
