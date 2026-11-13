//go:build notmuch

package lib

import "github.com/seamanticus/aerc/lib/notmuch"

func NotmuchVersion() (string, bool) {
	return notmuch.Version(), true
}
