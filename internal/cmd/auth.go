package cmd

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/builtbyrobben/quickbooks-cli/internal/api"
	"github.com/builtbyrobben/quickbooks-cli/internal/outfmt"
	"github.com/builtbyrobben/quickbooks-cli/internal/quickbooks"
	"github.com/builtbyrobben/quickbooks-cli/internal/secrets"
)

const sourceEnv = "env"

type AuthCmd struct {
	Login          AuthLoginCmd          `cmd:"" help:"Authenticate via OAuth 2.0 flow"`
	SetCredentials AuthSetCredentialsCmd `cmd:"" help:"Set OAuth client ID and secret"`
	SetRealm       AuthSetRealmCmd       `cmd:"" help:"Set QuickBooks company/realm ID"`
	Status         AuthStatusCmd         `cmd:"" help:"Show authentication status"`
	Remove         AuthRemoveCmd         `cmd:"" help:"Remove all stored credentials"`
}

// AuthLoginCmd handles the OAuth 2.0 authorization code flow.
type AuthLoginCmd struct{}

func (cmd *AuthLoginCmd) Run(ctx context.Context) error {
	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	clientID, err := resolveCredential(store, secrets.KeyClientID, "QUICKBOOKS_CLIENT_ID")
	if err != nil {
		return fmt.Errorf("client ID not set: run 'quickbooks-cli auth set-credentials' first: %w", err)
	}

	clientSecret, err := resolveCredential(store, secrets.KeyClientSecret, "QUICKBOOKS_CLIENT_SECRET")
	if err != nil {
		return fmt.Errorf("client secret not set: run 'quickbooks-cli auth set-credentials' first: %w", err)
	}

	// Build the authorization URL
	params := url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"scope":         {quickbooks.DefaultScopes()},
		"redirect_uri":  {quickbooks.DefaultRedirectURI()},
		"state":         {"quickbooks-cli"},
	}
	authURL := quickbooks.AuthURL() + "?" + params.Encode()

	fmt.Fprintln(os.Stderr, "Open this URL in your browser to authorize:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, authURL)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "After authorizing, paste the full redirect URL here:")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("read redirect URL: %w", scanner.Err())
	}

	redirectURL := strings.TrimSpace(scanner.Text())

	parsed, err := url.Parse(redirectURL)
	if err != nil {
		return fmt.Errorf("parse redirect URL: %w", err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		return fmt.Errorf("no authorization code found in URL")
	}

	realmID := parsed.Query().Get("realmId")
	if realmID != "" {
		if storeErr := store.SetCredential(secrets.KeyRealmID, realmID); storeErr != nil {
			return fmt.Errorf("store realm ID: %w", storeErr)
		}

		fmt.Fprintf(os.Stderr, "Realm ID: %s (stored)\n", realmID)
	}

	// Exchange authorization code for tokens
	tokenResp, err := exchangeAuthCode(ctx, clientID, clientSecret, code)
	if err != nil {
		return fmt.Errorf("exchange authorization code: %w", err)
	}

	if storeErr := store.SetCredential(secrets.KeyAccessToken, tokenResp.AccessToken); storeErr != nil {
		return fmt.Errorf("store access token: %w", storeErr)
	}

	if tokenResp.RefreshToken != "" {
		if storeErr := store.SetCredential(secrets.KeyRefreshToken, tokenResp.RefreshToken); storeErr != nil {
			return fmt.Errorf("store refresh token: %w", storeErr)
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":  "success",
			"message": "OAuth tokens stored in keyring",
		})
	}

	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout, []string{"STATUS", "MESSAGE"}, [][]string{{"success", "OAuth tokens stored in keyring"}})
	}

	fmt.Fprintln(os.Stderr, "OAuth tokens stored in keyring. You are now authenticated.")

	return nil
}

func exchangeAuthCode(ctx context.Context, clientID, clientSecret, code string) (*api.TokenResponse, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {quickbooks.DefaultRedirectURI()},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, quickbooks.TokenURL(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	credentials := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp api.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &tokenResp, nil
}

// AuthSetCredentialsCmd sets OAuth client ID and secret.
type AuthSetCredentialsCmd struct{}

func (cmd *AuthSetCredentialsCmd) Run(ctx context.Context) error {
	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	clientID, err := readSecret("Enter Client ID: ")
	if err != nil {
		return fmt.Errorf("read client ID: %w", err)
	}

	clientSecret, err := readSecret("Enter Client Secret: ")
	if err != nil {
		return fmt.Errorf("read client secret: %w", err)
	}

	if err := store.SetCredential(secrets.KeyClientID, clientID); err != nil {
		return fmt.Errorf("store client ID: %w", err)
	}

	if err := store.SetCredential(secrets.KeyClientSecret, clientSecret); err != nil {
		return fmt.Errorf("store client secret: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":  "success",
			"message": "OAuth credentials stored in keyring",
		})
	}

	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout, []string{"STATUS", "MESSAGE"}, [][]string{{"success", "OAuth credentials stored in keyring"}})
	}

	fmt.Fprintln(os.Stderr, "OAuth credentials stored in keyring")

	return nil
}

// AuthSetRealmCmd stores the realm/company ID.
type AuthSetRealmCmd struct {
	ID string `arg:"" required:"" help:"QuickBooks company/realm ID"`
}

func (cmd *AuthSetRealmCmd) Run(ctx context.Context) error {
	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	if err := store.SetCredential(secrets.KeyRealmID, cmd.ID); err != nil {
		return fmt.Errorf("store realm ID: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":  "success",
			"message": "Realm ID stored in keyring",
		})
	}

	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout, []string{"STATUS", "MESSAGE"}, [][]string{{"success", "Realm ID stored in keyring"}})
	}

	fmt.Fprintln(os.Stderr, "Realm ID stored in keyring")

	return nil
}

// AuthStatusCmd shows authentication status.
type AuthStatusCmd struct{}

func (cmd *AuthStatusCmd) Run(ctx context.Context) error {
	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	status := map[string]any{
		"storage_backend": "keyring",
	}

	// Check each credential
	for _, key := range []string{
		secrets.KeyClientID,
		secrets.KeyClientSecret,
		secrets.KeyRefreshToken,
		secrets.KeyRealmID,
		secrets.KeyAccessToken,
	} {
		has, _ := store.HasCredential(key)
		envOverride := getEnvOverride(key) != ""
		status["has_"+key] = has || envOverride

		if envOverride {
			status[key+"_source"] = sourceEnv
		} else if has {
			status[key+"_source"] = "keyring"
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, status)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"CLIENT_ID", "CLIENT_SECRET", "REFRESH_TOKEN", "REALM_ID", "ACCESS_TOKEN", "STORAGE"}

		credVal := func(key string) string {
			has, _ := status["has_"+key].(bool)
			source, _ := status[key+"_source"].(string)

			if source == sourceEnv {
				return sourceEnv
			}

			if has {
				return "set"
			}

			return "missing"
		}

		rows := [][]string{{
			credVal(secrets.KeyClientID),
			credVal(secrets.KeyClientSecret),
			credVal(secrets.KeyRefreshToken),
			credVal(secrets.KeyRealmID),
			credVal(secrets.KeyAccessToken),
			"keyring",
		}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	// Human-readable output
	fmt.Fprintf(os.Stdout, "Storage: %s\n\n", status["storage_backend"])

	printCredStatus := func(label, key string) {
		has, _ := status["has_"+key].(bool)
		source, _ := status[key+"_source"].(string)

		switch {
		case source == sourceEnv:
			fmt.Fprintf(os.Stdout, "  %-16s  set (env override)\n", label+":")
		case has:
			val, valErr := store.GetCredential(key)
			if valErr == nil && len(val) > 8 {
				fmt.Fprintf(os.Stdout, "  %-16s  %s...%s\n", label+":", val[:4], val[len(val)-4:])
			} else {
				fmt.Fprintf(os.Stdout, "  %-16s  set\n", label+":")
			}
		default:
			fmt.Fprintf(os.Stdout, "  %-16s  not set\n", label+":")
		}
	}

	printCredStatus("Client ID", secrets.KeyClientID)
	printCredStatus("Client Secret", secrets.KeyClientSecret)
	printCredStatus("Refresh Token", secrets.KeyRefreshToken)
	printCredStatus("Realm ID", secrets.KeyRealmID)
	printCredStatus("Access Token", secrets.KeyAccessToken)

	return nil
}

// AuthRemoveCmd removes all stored credentials.
type AuthRemoveCmd struct{}

func (cmd *AuthRemoveCmd) Run(ctx context.Context) error {
	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	if err := store.DeleteAll(); err != nil {
		return fmt.Errorf("remove credentials: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":  "success",
			"message": "All credentials removed",
		})
	}

	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout, []string{"STATUS", "MESSAGE"}, [][]string{{"success", "All credentials removed"}})
	}

	fmt.Fprintln(os.Stderr, "All credentials removed")

	return nil
}

// readSecret prompts for a secret value, hiding input if on a TTY.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		byteVal, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)

		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}

		return strings.TrimSpace(string(byteVal)), nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("read input: %w", scanner.Err())
	}

	return strings.TrimSpace(scanner.Text()), nil
}

// resolveCredential checks env var override, then keyring.
func resolveCredential(store secrets.Store, key, envVar string) (string, error) {
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}

	return store.GetCredential(key)
}

// getEnvOverride returns the env var value for a given credential key.
func getEnvOverride(key string) string {
	envMap := map[string]string{
		secrets.KeyClientID:     "QUICKBOOKS_CLIENT_ID",
		secrets.KeyClientSecret: "QUICKBOOKS_CLIENT_SECRET",
		secrets.KeyRefreshToken: "QUICKBOOKS_REFRESH_TOKEN",
		secrets.KeyRealmID:      "QUICKBOOKS_REALM_ID",
	}

	if envVar, ok := envMap[key]; ok {
		return os.Getenv(envVar)
	}

	return ""
}
