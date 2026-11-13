# aerc

![logo](contrib/logo.svg)

[![CI](https://github.com/seamanticus/aerc/actions/workflows/ci.yml/badge.svg)](https://github.com/seamanticus/aerc/actions/workflows/ci.yml)

aerc is an email client for your terminal.

This is a hard fork of [aerc by Robin Jarry](https://sr.ht/~rjarry/aerc/),
itself a fork of [the original aerc](https://git.sr.ht/~sircmpwn/aerc) by
Drew DeVault. It does not track or send patches upstream, and upstream
support channels do not cover it. It drops the notmuch backend and adds
OAuth2 login with encrypted token storage (`:oauth2-login`).

## Usage

On its first run, aerc will copy the default config files to `~/.config/aerc`
on Linux or `~/Library/Preferences/aerc` on MacOS (or `$XDG_CONFIG_HOME/aerc` if set)
and show the account configuration wizard.

If you redirect stdout to a file, logging output will be written to that file:

    $ aerc > log

Note that the default HTML filter additionally needs `w3m` to be installed
along with optional `unshare` (from `util-linux`) or `socksify` (from
`dante-utils`).

### Documentation

Also available as man pages:

- [aerc(1)](doc/aerc.1.scd)
- [aerc-accounts(5)](doc/aerc-accounts.5.scd)
- [aerc-binds(5)](doc/aerc-binds.5.scd)
- [aerc-config(5)](doc/aerc-config.5.scd)
- [aerc-imap(5)](doc/aerc-imap.5.scd)
- [aerc-jmap(5)](doc/aerc-jmap.5.scd)
- [aerc-maildir(5)](doc/aerc-maildir.5.scd)
- [aerc-mbox(5)](doc/aerc-mbox.5.scd)
- [aerc-patch(7)](doc/aerc-patch.7.scd)
- [aerc-search(1)](doc/aerc-search.1.scd)
- [aerc-sendmail(5)](doc/aerc-sendmail.5.scd)
- [aerc-smtp(5)](doc/aerc-smtp.5.scd)
- [aerc-stylesets(7)](doc/aerc-stylesets.7.scd)
- [aerc-templates(7)](doc/aerc-templates.7.scd)
- [aerc-tutorial(7)](doc/aerc-tutorial.7.scd)
- [carddav-query(1)](doc/carddav-query.1.scd)

Integration recipes for external tools (written for upstream aerc, most apply):

- [wiki](https://man.sr.ht/~rjarry/aerc/)

## Installation

Distribution packages named aerc ship upstream aerc, not this fork; build it
from source.

### From Source

Install the dependencies:

- go (>=1.27)
- [scdoc](https://git.sr.ht/~sircmpwn/scdoc)
- GNU make
- a C compiler, for the `colorize` and `wrap` filters

Then compile aerc:

    $ gmake

To install aerc locally (no need to be root):

    $ gmake install PREFIX=~/.local
    $ mandb -q ~/.local/share/man

By default, aerc will install config files to directories under `/usr/local/aerc`,
and will search for templates and stylesets in these locations in order:

- `${XDG_CONFIG_HOME:-~/.config}/aerc`
- `${XDG_DATA_HOME:-~/.local/share}/aerc`
- `/usr/local/share/aerc`
- `/usr/share/aerc`

At build time it is possible to add an extra location to this list and to use
that location as the default install location for config files by setting the
`PREFIX` option like so:

    # gmake PREFIX=/custom/location
    # gmake install PREFIX=/custom/location
    # mandb -q /custom/location/share/man

This will install templates and other config files to `/custom/location/share/aerc`,
and man pages to `/custom/location/share/man`. This extra location will have lower
priority than the XDG locations but higher than the fixed paths.

## Issues

Report bugs and feature requests at
[github.com/seamanticus/aerc/issues](https://github.com/seamanticus/aerc/issues).

Report security vulnerabilities privately through
[GitHub security advisories](https://github.com/seamanticus/aerc/security/advisories/new),
not in public issues.

[License](LICENSE).
