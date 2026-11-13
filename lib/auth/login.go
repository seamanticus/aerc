package auth

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"

	"github.com/seamanticus/aerc/lib/log"
	"golang.org/x/oauth2"
)

// BrowserLogin performs an authorization code grant with PKCE (RFC 7636)
// using a loopback redirect URI (RFC 8252). open is called with the URL to
// visit in a web browser.
func BrowserLogin(
	ctx context.Context, o *oauth2.Config, loginHint string,
	open func(string) error,
) (*oauth2.Token, error) {
	if o.Endpoint.AuthURL == "" {
		return nil, errors.New("auth_endpoint is not set")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer func() { _ = l.Close() }()

	conf := *o
	conf.RedirectURL = fmt.Sprintf("http://%s/", l.Addr())
	verifier := oauth2.GenerateVerifier()
	state := oauth2.GenerateVerifier()
	opts := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(verifier)}
	if loginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", loginHint))
	}

	type result struct {
		code string
		err  error
	}
	results := make(chan result, 1)
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if q.Get("state") != state {
				// unrelated request (favicon, other local process)
				http.Error(w, "invalid state", http.StatusBadRequest)
				return
			}
			var res result
			if e := q.Get("error"); e != "" {
				res.err = fmt.Errorf("authorization failed: %s: %s",
					e, q.Get("error_description"))
			} else {
				res.code = q.Get("code")
			}
			// the browser page is informational only
			if res.err != nil {
				_, _ = fmt.Fprintf(w, "aerc: %s\n", html.EscapeString(res.err.Error()))
			} else {
				_, _ = fmt.Fprintln(w, "aerc: authorization complete, you may close this window.")
			}
			select {
			case results <- res:
			default:
			}
		}),
	}
	go func() {
		defer log.PanicHandler()
		_ = srv.Serve(l)
	}()
	defer func() { _ = srv.Close() }()

	if err := open(conf.AuthCodeURL(state, opts...)); err != nil {
		return nil, err
	}

	var res result
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res = <-results:
	}
	if res.err != nil {
		return nil, res.err
	}
	return conf.Exchange(ctx, res.code, oauth2.VerifierOption(verifier))
}

// DeviceLogin performs a device authorization grant (RFC 8628). prompt is
// called with the verification URI and the code the user must enter there.
func DeviceLogin(
	ctx context.Context, o *oauth2.Config, prompt func(uri, code string),
) (*oauth2.Token, error) {
	if o.Endpoint.DeviceAuthURL == "" {
		return nil, errors.New("device_endpoint is not set")
	}
	da, err := o.DeviceAuth(ctx)
	if err != nil {
		return nil, err
	}
	uri := da.VerificationURIComplete
	if uri == "" {
		uri = da.VerificationURI
	}
	prompt(uri, da.UserCode)
	return o.DeviceAccessToken(ctx, da)
}

// LoginHint returns the user name of an URI, used to preselect the account
// on the provider's login page.
func LoginHint(uri *url.URL) string {
	if uri.User == nil {
		return ""
	}
	return uri.User.Username()
}
