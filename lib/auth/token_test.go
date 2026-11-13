package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type fakeProvider struct {
	*httptest.Server
	refreshes atomic.Int32
	// refresh token accepted by the token endpoint
	valid string
	// refresh token returned on refresh, rotation when != valid
	next      string
	challenge string
}

func newFakeProvider(t *testing.T, valid, next string) *fakeProvider {
	p := &fakeProvider{valid: valid, next: next}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		reply := func(code int, v map[string]any) {
			w.WriteHeader(code)
			_ = json.NewEncoder(w).Encode(v)
		}
		switch r.Form.Get("grant_type") {
		case "refresh_token":
			p.refreshes.Add(1)
			if r.Form.Get("refresh_token") != p.valid {
				reply(400, map[string]any{
					"error":             "invalid_grant",
					"error_description": "AADSTS70008: expired",
				})
				return
			}
		case "authorization_code":
			if r.Form.Get("code") != "the-code" ||
				oauth2.S256ChallengeFromVerifier(r.Form.Get("code_verifier")) != p.challenge {
				reply(400, map[string]any{"error": "invalid_grant"})
				return
			}
		case "urn:ietf:params:oauth:grant-type:device_code":
			if r.Form.Get("device_code") != "dev-code" {
				reply(400, map[string]any{"error": "invalid_grant"})
				return
			}
		default:
			reply(400, map[string]any{"error": "unsupported_grant_type"})
			return
		}
		reply(200, map[string]any{
			"access_token":  "access-" + p.next,
			"refresh_token": p.next,
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	})
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code":      "dev-code",
			"user_code":        "ABCD1234",
			"verification_uri": "https://example.invalid/link",
			"expires_in":       60,
			"interval":         1,
		})
	})
	p.Server = httptest.NewServer(mux)
	t.Cleanup(p.Close)
	return p
}

func (p *fakeProvider) config() *oauth2.Config {
	return &oauth2.Config{
		ClientID: "client",
		Scopes:   []string{"offline_access", "imap"},
		Endpoint: oauth2.Endpoint{
			AuthURL:       p.URL + "/authorize",
			DeviceAuthURL: p.URL + "/device",
			TokenURL:      p.URL + "/token",
		},
	}
}

func newTestStore(t *testing.T) *TokenStore {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	return &TokenStore{
		Account:    t.Name(),
		EncryptCmd: "base64",
		DecryptCmd: "base64 -d",
	}
}

func TestGetAccessTokenCachesInMemory(t *testing.T) {
	p := newFakeProvider(t, "r1", "r1")
	s := newTestStore(t)
	if err := s.save("r1"); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		tok, err := GetAccessToken(p.config(), s, "")
		if err != nil {
			t.Fatal(err)
		}
		if tok != "access-r1" {
			t.Fatalf("got access token %q", tok)
		}
	}
	if n := p.refreshes.Load(); n != 1 {
		t.Fatalf("expected 1 refresh, got %d", n)
	}
}

func TestGetAccessTokenSavesRotatedToken(t *testing.T) {
	p := newFakeProvider(t, "r1", "r2")
	s := newTestStore(t)
	if err := s.save("r1"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetAccessToken(p.config(), s, ""); err != nil {
		t.Fatal(err)
	}
	r, err := s.load()
	if err != nil {
		t.Fatal(err)
	}
	if r != "r2" {
		t.Fatalf("stored refresh token %q, expected r2", r)
	}
	buf, err := os.ReadFile(s.path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(buf), "r2") {
		t.Fatal("refresh token stored in clear text")
	}
}

func TestGetAccessTokenRejectedRequiresLogin(t *testing.T) {
	p := newFakeProvider(t, "r1", "r1")
	s := newTestStore(t)
	if err := s.save("revoked"); err != nil {
		t.Fatal(err)
	}
	_, err := GetAccessToken(p.config(), s, "")
	if !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("expected ErrLoginRequired, got %v", err)
	}
	// x/oauth2 retries a failed request with another client auth style,
	// so a single rejected refresh may issue more than one request
	sent := p.refreshes.Load()
	// reconnect attempts must not hit the provider again
	_, err = GetAccessToken(p.config(), s, "")
	if !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("expected ErrLoginRequired, got %v", err)
	}
	if n := p.refreshes.Load(); n != sent {
		t.Fatalf("expected no new refresh request, got %d", n-sent)
	}
	// a new login clears the condition
	err = s.Store(&oauth2.Token{AccessToken: "a", RefreshToken: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GetAccessToken(p.config(), s, ""); err != nil {
		t.Fatal(err)
	}
}

func TestGetAccessTokenNoTokenRequiresLogin(t *testing.T) {
	p := newFakeProvider(t, "r1", "r1")
	s := newTestStore(t)
	// an unencrypted token at the location used by previous versions
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	old := filepath.Join(os.Getenv("XDG_CACHE_HOME"), "aerc", s.Account+"-xoauth2.token")
	if err := os.MkdirAll(filepath.Dir(old), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("r1"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := GetAccessToken(p.config(), s, "")
	if !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("expected ErrLoginRequired, got %v", err)
	}
	if n := p.refreshes.Load(); n != 0 {
		t.Fatalf("expected no refresh request, got %d", n)
	}
}

func TestGetAccessTokenRefreshTimeout(t *testing.T) {
	stall := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	t.Cleanup(stall.Close)
	old := refreshTimeout
	refreshTimeout = 10 * time.Millisecond
	t.Cleanup(func() { refreshTimeout = old })

	s := newTestStore(t)
	if err := s.save("r1"); err != nil {
		t.Fatal(err)
	}
	// rotate r1 -> r2 without saving, so only memory holds r2
	noSave := *s
	noSave.EncryptCmd = ""
	if _, err := GetAccessToken(newFakeProvider(t, "r1", "r2").config(), &noSave, ""); err != nil {
		t.Fatal(err)
	}
	cacheEntry(s.Account).token.Expiry = time.Now().Add(-time.Hour)

	o := newFakeProvider(t, "r2", "r2").config()
	stalled := *o
	stalled.Endpoint.TokenURL = stall.URL
	_, err := GetAccessToken(&stalled, s, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected a deadline error, got %v", err)
	}
	// a stalled endpoint is transient: retry with the rotated token
	if tok, err := GetAccessToken(o, s, ""); err != nil || tok != "access-r2" {
		t.Fatalf("expected access-r2 after timeout, got %q, %v", tok, err)
	}
}

func TestGetAccessTokenWithoutStorageUsesPassword(t *testing.T) {
	p := newFakeProvider(t, "r1", "r2")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	s := &TokenStore{Account: t.Name()}
	tok, err := GetAccessToken(p.config(), s, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "access-r2" {
		t.Fatalf("got access token %q", tok)
	}
	if _, err := os.Stat(s.path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("token file must not be written without encryption: %v", err)
	}
}

func TestStoreRequiresEncryption(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	s := &TokenStore{Account: t.Name(), DecryptCmd: "cat"}
	err := s.Store(&oauth2.Token{AccessToken: "a", RefreshToken: "r"})
	if err == nil {
		t.Fatal("expected an error without oauth2-encrypt-cmd")
	}
}

func TestBrowserLogin(t *testing.T) {
	p := newFakeProvider(t, "r1", "r1")
	open := func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		if q.Get("code_challenge_method") != "S256" {
			return fmt.Errorf("PKCE not used: %s", authURL)
		}
		if q.Get("login_hint") != "user@example.com" {
			return fmt.Errorf("missing login_hint: %s", authURL)
		}
		p.challenge = q.Get("code_challenge")
		// simulate the provider redirecting the browser
		go func() {
			cb := q.Get("redirect_uri") + "?code=the-code&state=" +
				url.QueryEscape(q.Get("state"))
			resp, err := http.Get(cb)
			if err != nil {
				t.Error(err)
				return
			}
			_ = resp.Body.Close()
		}()
		return nil
	}
	tok, err := BrowserLogin(context.Background(), p.config(), "user@example.com", open)
	if err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken != "r1" {
		t.Fatalf("got refresh token %q", tok.RefreshToken)
	}
}

func TestBrowserLoginIgnoresWrongState(t *testing.T) {
	p := newFakeProvider(t, "r1", "r1")
	ctx, cancel := context.WithCancel(context.Background())
	open := func(authURL string) error {
		u, _ := url.Parse(authURL)
		q := u.Query()
		go func() {
			defer cancel()
			resp, err := http.Get(q.Get("redirect_uri") + "?code=the-code&state=forged")
			if err != nil {
				t.Error(err)
				return
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("forged state answered with %d", resp.StatusCode)
			}
		}()
		return nil
	}
	_, err := BrowserLogin(ctx, p.config(), "", open)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestOAuth2ConfigProviderDefaults(t *testing.T) {
	ms := microsoftDefaults.Endpoint.TokenURL
	vectors := []struct {
		uri      string
		clientID string
		tokenURL string
	}{
		// no parameters: everything from the catalogue
		{"imaps+xoauth2://u%40hotmail.com@outlook.office365.com:993", thunderbirdMicrosoftClientID, ms},
		{"smtp+xoauth2://u%40hotmail.com@smtp-mail.outlook.com:587", thunderbirdMicrosoftClientID, ms},
		{"smtp+xoauth2://u%40hotmail.com@smtp.office365.com:587", thunderbirdMicrosoftClientID, ms},
		{"imaps+xoauth2://u%40hotmail.com@imap-mail.outlook.com", thunderbirdMicrosoftClientID, ms},
		// explicit parameters win, missing ones are filled
		{"imaps+xoauth2://u@outlook.office365.com?client_id=mine", "mine", ms},
		{"imaps+xoauth2://u@outlook.office365.com?token_endpoint=https://t", thunderbirdMicrosoftClientID, "https://t"},
		// password without token_endpoint is an access token: untouched
		{"imaps+xoauth2://u:access@outlook.office365.com", "", ""},
		// password with token_endpoint is a refresh token: completed
		{"imaps+xoauth2://u:refresh@outlook.office365.com?token_endpoint=https://t", thunderbirdMicrosoftClientID, "https://t"},
		// unknown host
		{"imaps+xoauth2://u@imap.example.com", "", ""},
	}
	for _, v := range vectors {
		u, err := url.Parse(v.uri)
		if err != nil {
			t.Fatal(err)
		}
		o := OAuth2Config(u)
		if o.ClientID != v.clientID || o.Endpoint.TokenURL != v.tokenURL {
			t.Errorf("%s: got client_id=%q token_endpoint=%q, expected %q %q",
				v.uri, o.ClientID, o.Endpoint.TokenURL, v.clientID, v.tokenURL)
		}
	}
}

func TestDeviceLogin(t *testing.T) {
	p := newFakeProvider(t, "r1", "r1")
	var code string
	tok, err := DeviceLogin(context.Background(), p.config(), func(uri, c string) {
		code = c
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != "ABCD1234" {
		t.Fatalf("got user code %q", code)
	}
	if tok.RefreshToken != "r1" {
		t.Fatalf("got refresh token %q", tok.RefreshToken)
	}
}
