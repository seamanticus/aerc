package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/seamanticus/aerc/config"
	"github.com/seamanticus/aerc/lib/log"
	"github.com/seamanticus/aerc/lib/xdg"
	"golang.org/x/oauth2"
)

var ErrLoginRequired = errors.New(
	"new OAuth2 authentication required, run :oauth2-login")

// TokenStore holds the OAuth2 tokens of one account. The refresh token is
// persisted encrypted with user supplied commands. Access tokens are only
// kept in memory.
type TokenStore struct {
	Account    string
	EncryptCmd string
	DecryptCmd string
}

func NewTokenStore(acct *config.AccountConfig) *TokenStore {
	return &TokenStore{
		Account:    acct.Name,
		EncryptCmd: acct.OAuth2EncryptCmd,
		DecryptCmd: acct.OAuth2DecryptCmd,
	}
}

// IMAP and SMTP authenticate from different goroutines with separate
// TokenStore values. The cache is keyed by account name so that both share
// the same tokens and never refresh concurrently.
type cachedToken struct {
	sync.Mutex
	token *oauth2.Token
	// set when the provider rejected the refresh token, so that reconnect
	// attempts do not run oauth2-decrypt-cmd (and pinentry) again
	loginRequired bool
}

var (
	cacheLock sync.Mutex
	cache     = make(map[string]*cachedToken)
)

func cacheEntry(account string) *cachedToken {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	e, ok := cache[account]
	if !ok {
		e = new(cachedToken)
		cache[account] = e
	}
	return e
}

func (s *TokenStore) path() string {
	return xdg.StatePath("aerc", "oauth2", s.Account+".token")
}

func (s *TokenStore) run(command string, input []byte, pinentry bool) ([]byte, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if pinentry && config.EnablePinentry != nil &&
		config.DisablePinentry != nil && config.SetPinentryEnv != nil {
		config.EnablePinentry()
		defer config.DisablePinentry()
		config.SetPinentryEnv(cmd)
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s",
			command, err, strings.TrimSpace(stderr.String()))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: empty output", command)
	}
	return out, nil
}

// Returns an error wrapping fs.ErrNotExist when no token was saved or when no
// decryption command is configured.
func (s *TokenStore) load() (string, error) {
	if s.DecryptCmd == "" {
		return "", fs.ErrNotExist
	}
	buf, err := os.ReadFile(s.path())
	if err != nil {
		return "", err
	}
	// gpg may need a passphrase to decrypt
	out, err := s.run(s.DecryptCmd, buf, true)
	if err != nil {
		return "", fmt.Errorf("oauth2-decrypt-cmd: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *TokenStore) save(refreshToken string) error {
	if s.EncryptCmd == "" {
		return errors.New("oauth2-encrypt-cmd is not set")
	}
	out, err := s.run(s.EncryptCmd, []byte(refreshToken), false)
	if err != nil {
		return fmt.Errorf("oauth2-encrypt-cmd: %w", err)
	}
	p := s.path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	// write then rename so that a failure never leaves a truncated token
	tmp, err := os.CreateTemp(filepath.Dir(p), ".token-*")
	if err != nil {
		return err
	}
	// fails with ErrNotExist once renamed
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

// Store saves the tokens obtained by a new authorization. Both encryption
// commands are required so that the token can be read back later.
func (s *TokenStore) Store(tok *oauth2.Token) error {
	if s.EncryptCmd == "" || s.DecryptCmd == "" {
		return errors.New(
			"oauth2-encrypt-cmd and oauth2-decrypt-cmd must be set to store OAuth2 tokens")
	}
	if tok.RefreshToken == "" {
		return errors.New("no refresh token received, is offline_access in scope?")
	}
	e := cacheEntry(s.Account)
	e.Lock()
	defer e.Unlock()
	if err := s.save(tok.RefreshToken); err != nil {
		return err
	}
	e.token = tok
	e.loginRequired = false
	return nil
}

func exchangeRefreshToken(o *oauth2.Config, refreshToken string) (*oauth2.Token, error) {
	token := new(oauth2.Token)
	token.RefreshToken = refreshToken
	token.TokenType = "Bearer"
	return o.TokenSource(context.TODO(), token).Token()
}

// GetAccessToken returns a valid access token for the account. When no token
// endpoint is configured, password is the access token. Otherwise, password
// is used as a refresh token if none was stored by :oauth2-login.
func GetAccessToken(
	o *oauth2.Config, store *TokenStore, password string,
) (string, error) {
	if o.Endpoint.TokenURL == "" {
		return password, nil
	}
	e := cacheEntry(store.Account)
	e.Lock()
	defer e.Unlock()

	if e.token.Valid() {
		return e.token.AccessToken, nil
	}
	if e.loginRequired {
		return "", ErrLoginRequired
	}

	var refreshToken string
	if e.token != nil {
		refreshToken = e.token.RefreshToken
	}
	if refreshToken == "" {
		r, err := store.load()
		switch {
		case err == nil:
			refreshToken = r
		case errors.Is(err, fs.ErrNotExist):
			refreshToken = password
		default:
			return "", err
		}
	}
	if refreshToken == "" {
		return "", fmt.Errorf("no OAuth2 token: %w", ErrLoginRequired)
	}

	token, err := exchangeRefreshToken(o, refreshToken)
	if err != nil {
		e.token = nil
		var rerr *oauth2.RetrieveError
		if errors.As(err, &rerr) && rerr.ErrorCode == "invalid_grant" {
			e.loginRequired = true
			return "", fmt.Errorf("refresh token rejected (%s): %w",
				rerr.ErrorDescription, ErrLoginRequired)
		}
		return "", err
	}
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	if token.RefreshToken != refreshToken && store.EncryptCmd != "" {
		// The previous refresh token remains in memory and on disk and
		// stays usable for some time: do not fail the connection.
		if err := store.save(token.RefreshToken); err != nil {
			log.Errorf("%s: failed to save refresh token: %v",
				store.Account, err)
		}
	}
	e.token = token
	return token.AccessToken, nil
}
