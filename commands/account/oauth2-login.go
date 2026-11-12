package account

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/auth"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"golang.org/x/oauth2"
)

type OAuth2Login struct {
	Device bool `opt:"-d" desc:"Use the device authorization flow."`
}

func init() {
	commands.Register(OAuth2Login{})
}

func (OAuth2Login) Description() string {
	return "Authorize the current account with its OAuth2 provider."
}

func (OAuth2Login) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (OAuth2Login) Aliases() []string {
	return []string{"oauth2-login"}
}

// Returns the first of source or outgoing URIs that uses OAuth2 with a token
// endpoint. Both share the same stored token.
func oauth2URI(acct *config.AccountConfig) (*url.URL, error) {
	outgoing, err := acct.Outgoing.ConnectionString()
	if err != nil {
		return nil, err
	}
	for _, s := range []string{acct.Source, outgoing} {
		u, err := url.Parse(s)
		if err != nil {
			continue
		}
		oauth := strings.HasSuffix(u.Scheme, "+oauthbearer") ||
			strings.HasSuffix(u.Scheme, "+xoauth2")
		if oauth && auth.OAuth2Config(u).Endpoint.TokenURL != "" {
			return u, nil
		}
	}
	return nil, errors.New("no source or outgoing using OAuth2 with a token_endpoint")
}

func (l OAuth2Login) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	conf := acct.AccountConfig()
	store := auth.NewTokenStore(conf)
	if store.EncryptCmd == "" || store.DecryptCmd == "" {
		return errors.New("oauth2-encrypt-cmd and oauth2-decrypt-cmd must be set")
	}
	u, err := oauth2URI(conf)
	if err != nil {
		return err
	}
	o := auth.OAuth2Config(u)

	go func() {
		defer log.PanicHandler()
		// matches the lifetime of a Microsoft device code
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		tok, err := l.login(ctx, o, u)
		if err == nil {
			err = store.Store(tok)
		}
		if err != nil {
			acct.PushError(fmt.Errorf("oauth2-login: %w", err))
			return
		}
		app.PushSuccess(conf.Name + ": OAuth2 authorization complete, run :connect")
	}()
	return nil
}

func (l OAuth2Login) login(
	ctx context.Context, o *oauth2.Config, u *url.URL,
) (*oauth2.Token, error) {
	if l.Device {
		return auth.DeviceLogin(ctx, o, func(uri, code string) {
			msg := fmt.Sprintf("oauth2-login: open %s and enter code %s", uri, code)
			log.Infof("%s", msg)
			app.PushStatus(msg, 15*time.Minute)
		})
	}
	return auth.BrowserLogin(ctx, o, auth.LoginHint(u), func(authURL string) error {
		log.Infof("oauth2-login: opening %s", authURL)
		app.PushStatus("oauth2-login: waiting for authorization in web browser", 15*time.Minute)
		return lib.XDGOpenMime(authURL, "x-scheme-handler/https", "")
	})
}
