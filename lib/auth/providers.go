package auth

import (
	"net/url"

	"golang.org/x/oauth2"
)

// Mozilla Thunderbird's public client ID. Microsoft does not let personal
// accounts register applications without an Azure directory, so this is
// the client most open source mail programs use.
const thunderbirdMicrosoftClientID = "9e5f94bc-e8a4-4e73-b8be-63364c29d753"

// The common tenant accepts both personal and work/school accounts.
const microsoftAuthority = "https://login.microsoftonline.com/common/oauth2/v2.0"

var microsoftDefaults = oauth2.Config{
	ClientID: thunderbirdMicrosoftClientID,
	Scopes: []string{
		"offline_access",
		"https://outlook.office.com/IMAP.AccessAsUser.All",
		"https://outlook.office.com/SMTP.Send",
	},
	Endpoint: oauth2.Endpoint{
		AuthURL:       microsoftAuthority + "/authorize",
		DeviceAuthURL: microsoftAuthority + "/devicecode",
		TokenURL:      microsoftAuthority + "/token",
	},
}

// Mozilla Thunderbird's Google "installed application" client. Google's
// token endpoint rejects requests without the client secret even with PKCE;
// installed application secrets are not confidential and this one is
// published in Thunderbird's source.
const (
	thunderbirdGoogleClientID     = "406964657835-aq8lmia8j95dhl1a2bvharmfk3t1hgqj.apps.googleusercontent.com"
	thunderbirdGoogleClientSecret = "kSmqreRr0qwBWJgbf5Y-PjSU"
)

// Google returns a refresh token to installed applications without
// access_type=offline. Its device flow rejects this client type, so there
// is no DeviceAuthURL.
var googleDefaults = oauth2.Config{
	ClientID:     thunderbirdGoogleClientID,
	ClientSecret: thunderbirdGoogleClientSecret,
	Scopes:       []string{"https://mail.google.com/"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
	},
}

// OAuth2 parameters used for well known hosts when the URI omits them.
// The Microsoft aliases resolve to the same Exchange Online front ends as
// outlook.office365.com and smtp-mail.outlook.com; googlemail.com is the
// legacy name of gmail.com.
var providerDefaults = map[string]*oauth2.Config{
	"outlook.office365.com": &microsoftDefaults,
	"outlook.office.com":    &microsoftDefaults,
	"imap-mail.outlook.com": &microsoftDefaults,
	"imap.outlook.com":      &microsoftDefaults,
	"smtp-mail.outlook.com": &microsoftDefaults,
	"smtp.office365.com":    &microsoftDefaults,
	"smtp.outlook.com":      &microsoftDefaults,
	"imap.gmail.com":        &googleDefaults,
	"smtp.gmail.com":        &googleDefaults,
	"imap.googlemail.com":   &googleDefaults,
	"smtp.googlemail.com":   &googleDefaults,
}

// Fills the parameters missing from o with the defaults of the URI host.
// A URI with a password and no token_endpoint uses the password as access
// token (e.g. from an external helper in *-cred-cmd): it is left untouched.
func applyProviderDefaults(o *oauth2.Config, uri *url.URL) {
	d, ok := providerDefaults[uri.Hostname()]
	if !ok {
		return
	}
	if _, hasPassword := uri.User.Password(); hasPassword && o.Endpoint.TokenURL == "" {
		return
	}
	if o.ClientID == "" {
		// The secret belongs to the default client: a user supplied
		// client_id must come with its own client_secret, if any.
		o.ClientID = d.ClientID
		o.ClientSecret = d.ClientSecret
	}
	if len(o.Scopes) == 0 {
		o.Scopes = d.Scopes
	}
	if o.Endpoint.AuthURL == "" {
		o.Endpoint.AuthURL = d.Endpoint.AuthURL
	}
	if o.Endpoint.DeviceAuthURL == "" {
		o.Endpoint.DeviceAuthURL = d.Endpoint.DeviceAuthURL
	}
	if o.Endpoint.TokenURL == "" {
		o.Endpoint.TokenURL = d.Endpoint.TokenURL
	}
}
