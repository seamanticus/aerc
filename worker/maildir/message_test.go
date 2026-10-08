package maildir

import (
	"testing"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/models"
)

func TestMessageHeadersFlags(t *testing.T) {
	c, err := NewContainer(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	dir := c.Store.Dir("INBOX")
	if err := dir.Init(); err != nil {
		t.Fatal(err)
	}
	uid := addTestMessage(t, dir)
	if _, err := c.UIDs(dir); err != nil {
		t.Fatal(err)
	}
	m, err := c.Message(dir, uid)
	if err != nil {
		t.Fatal(err)
	}

	info, err := m.MessageHeaders()
	if err != nil {
		t.Fatal(err)
	}
	if info.Flags.Has(models.SeenFlag) {
		t.Fatal("new message is already seen")
	}

	// Headers are cached but flags must reflect the current state.
	if err := m.SetOneFlag(maildir.FlagSeen, true); err != nil {
		t.Fatal(err)
	}
	info, err = m.MessageHeaders()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Flags.Has(models.SeenFlag) {
		t.Fatal("cached headers returned stale flags")
	}

	// Callers must not be able to modify the cached headers.
	info.Flags = 0
	info, err = m.MessageHeaders()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Flags.Has(models.SeenFlag) {
		t.Fatal("cached headers were modified by caller")
	}
}
