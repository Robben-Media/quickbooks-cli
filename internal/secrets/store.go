package secrets

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"golang.org/x/term"

	"github.com/builtbyrobben/quickbooks-cli/internal/config"
)

// Store provides credential storage operations.
type Store interface {
	GetCredential(key string) (string, error)
	SetCredential(key, value string) error
	DeleteCredential(key string) error
	HasCredential(key string) (bool, error)
	DeleteAll() error
}

// KeyringStore implements Store using the system keyring.
type KeyringStore struct {
	ring keyring.Keyring
}

// Credential key constants for OAuth 2.0 storage.
const (
	KeyClientID          = "client_id"
	KeyClientSecret      = "client_secret"
	KeyRefreshToken      = "refresh_token"
	KeyRealmID           = "realm_id"
	KeyAccessToken       = "access_token"
	KeyAccessTokenExpiry = "access_token_expiry"
)

const (
	keyringPasswordEnv = "QUICKBOOKS_CLI_KEYRING_PASS" //nolint:gosec // env var name, not a credential
	keyringBackendEnv  = "QUICKBOOKS_CLI_KEYRING_BACKEND"
	keyringOpenTimeout = 5 * time.Second
)

var (
	errMissingCredentialKey  = errors.New("missing credential key")
	errMissingCredentialVal  = errors.New("credential value cannot be empty")
	errNoTTY                 = errors.New("no TTY available for keyring file backend password prompt")
	errInvalidKeyringBackend = errors.New("invalid keyring backend")
	errKeyringTimeout        = errors.New("keyring connection timed out")
)

// AllCredentialKeys returns all known credential keys for iteration.
func AllCredentialKeys() []string {
	return []string{
		KeyClientID,
		KeyClientSecret,
		KeyRefreshToken,
		KeyRealmID,
		KeyAccessToken,
		KeyAccessTokenExpiry,
	}
}

// KeyringBackendInfo describes which keyring backend is selected and why.
type KeyringBackendInfo struct {
	Value  string
	Source string
}

const (
	keyringBackendSourceEnv     = "env"
	keyringBackendSourceDefault = "default"
	keyringBackendAuto          = "auto"
)

// ResolveKeyringBackendInfo determines the keyring backend from environment.
func ResolveKeyringBackendInfo() (KeyringBackendInfo, error) {
	if v := normalizeKeyringBackend(os.Getenv(keyringBackendEnv)); v != "" {
		return KeyringBackendInfo{Value: v, Source: keyringBackendSourceEnv}, nil
	}

	return KeyringBackendInfo{Value: keyringBackendAuto, Source: keyringBackendSourceDefault}, nil
}

func allowedBackends(info KeyringBackendInfo) ([]keyring.BackendType, error) {
	switch info.Value {
	case "", keyringBackendAuto:
		return nil, nil
	case "keychain":
		return []keyring.BackendType{keyring.KeychainBackend}, nil
	case "file":
		return []keyring.BackendType{keyring.FileBackend}, nil
	default:
		return nil, fmt.Errorf("%w: %q (expected %s, keychain, or file)", errInvalidKeyringBackend, info.Value, keyringBackendAuto)
	}
}

func fileKeyringPasswordFunc() keyring.PromptFunc {
	return fileKeyringPasswordFuncFrom(os.Getenv(keyringPasswordEnv), term.IsTerminal(int(os.Stdin.Fd())))
}

func fileKeyringPasswordFuncFrom(password string, isTTY bool) keyring.PromptFunc {
	if password != "" {
		return keyring.FixedStringPrompt(password)
	}

	if isTTY {
		return keyring.TerminalPrompt
	}

	return func(_ string) (string, error) {
		return "", fmt.Errorf("%w; set %s", errNoTTY, keyringPasswordEnv)
	}
}

func normalizeKeyringBackend(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func shouldForceFileBackend(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == keyringBackendAuto && dbusAddr == ""
}

func shouldUseKeyringTimeout(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == "auto" && dbusAddr != ""
}

func openKeyring() (keyring.Keyring, error) {
	keyringDir, err := config.EnsureKeyringDir()
	if err != nil {
		return nil, fmt.Errorf("ensure keyring dir: %w", err)
	}

	backendInfo, err := ResolveKeyringBackendInfo()
	if err != nil {
		return nil, err
	}

	backends, err := allowedBackends(backendInfo)
	if err != nil {
		return nil, err
	}

	dbusAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if shouldForceFileBackend(runtime.GOOS, backendInfo, dbusAddr) {
		backends = []keyring.BackendType{keyring.FileBackend}
	}

	cfg := keyring.Config{
		ServiceName:              config.AppName,
		KeychainTrustApplication: false,
		AllowedBackends:          backends,
		FileDir:                  keyringDir,
		FilePasswordFunc:         fileKeyringPasswordFunc(),
	}

	if shouldUseKeyringTimeout(runtime.GOOS, backendInfo, dbusAddr) {
		return openKeyringWithTimeout(cfg, keyringOpenTimeout)
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return ring, nil
}

type keyringResult struct {
	ring keyring.Keyring
	err  error
}

func openKeyringWithTimeout(cfg keyring.Config, timeout time.Duration) (keyring.Keyring, error) {
	ch := make(chan keyringResult, 1)

	go func() {
		ring, err := keyring.Open(cfg)
		ch <- keyringResult{ring, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("open keyring: %w", res.err)
		}

		return res.ring, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("%w after %v (D-Bus SecretService may be unresponsive); "+
			"set QUICKBOOKS_CLI_KEYRING_BACKEND=file and QUICKBOOKS_CLI_KEYRING_PASS=<password> to use encrypted file storage instead",
			errKeyringTimeout, timeout)
	}
}

// OpenDefault opens the default credential store.
func OpenDefault() (Store, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}

	return &KeyringStore{ring: ring}, nil
}

// GetCredential retrieves a credential by key.
func (s *KeyringStore) GetCredential(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errMissingCredentialKey
	}

	item, err := s.ring.Get(key)
	if err != nil {
		return "", fmt.Errorf("read credential %q: %w", key, err)
	}

	return string(item.Data), nil
}

// SetCredential stores a credential by key.
func (s *KeyringStore) SetCredential(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errMissingCredentialKey
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return errMissingCredentialVal
	}

	if err := s.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(value),
	}); err != nil {
		return fmt.Errorf("store credential %q: %w", key, err)
	}

	return nil
}

// DeleteCredential removes a credential by key.
func (s *KeyringStore) DeleteCredential(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errMissingCredentialKey
	}

	if err := s.ring.Remove(key); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("delete credential %q: %w", key, err)
	}

	return nil
}

// HasCredential checks if a credential exists.
func (s *KeyringStore) HasCredential(key string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, errMissingCredentialKey
	}

	_, err := s.ring.Get(key)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("check credential %q: %w", key, err)
	}

	return true, nil
}

// DeleteAll removes all stored credentials.
func (s *KeyringStore) DeleteAll() error {
	for _, key := range AllCredentialKeys() {
		if err := s.ring.Remove(key); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
			return fmt.Errorf("delete credential %q: %w", key, err)
		}
	}

	return nil
}
