package imap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/seamanticus/aerc/lib/auth"
	"github.com/seamanticus/aerc/worker/types"
	"golang.org/x/oauth2"
)

func TestAuthError(t *testing.T) {
	retrieve := func(code int) error {
		return fmt.Errorf("oauth2: %w", &oauth2.RetrieveError{
			Response: &http.Response{StatusCode: code},
		})
	}
	vectors := []struct {
		name string
		err  error
		auth bool
	}{
		// the server or provider gave a verdict on the credentials
		{"imap NO", errors.New("AUTHENTICATE failed."), true},
		{"login required", fmt.Errorf("no OAuth2 token: %w", auth.ErrLoginRequired), true},
		{"token endpoint 400", retrieve(400), true},
		{"decrypt command", errors.New("oauth2-decrypt-cmd: exit status 2"), true},
		// no verdict: retrying cannot count as a failed login
		{"dial timeout", &net.OpError{Op: "dial", Err: os.ErrDeadlineExceeded}, false},
		{"refresh timeout", &url.Error{Op: "Post", URL: "https://t", Err: context.DeadlineExceeded}, false},
		{"eof", fmt.Errorf("read: %w", io.EOF), false},
		{"connection closed", errors.New(errMsgConnClosed), false},
		{"token endpoint 503", retrieve(503), false},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			err := authError(v.err)
			if got := errors.Is(err, types.ErrAuth); got != v.auth {
				t.Errorf("errors.Is(authError(%v), ErrAuth) = %t, want %t",
					v.err, got, v.auth)
			}
			if !errors.Is(err, v.err) {
				t.Errorf("authError(%v) lost the original error", v.err)
			}
		})
	}
}
