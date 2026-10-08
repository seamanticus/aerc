package maildir

import (
	"sync"
	"testing"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/models"
)

// TestMessageConcurrentAccess mimics the worker goroutine changing flags
// while the check-mail and sorting goroutines read the same cached messages.
// It is only meaningful with the race detector enabled (go test -race).
func TestMessageConcurrentAccess(t *testing.T) {
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

	// Errors are expected and ignored: files are renamed concurrently.
	const n = 200
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := range n {
			if m, err := c.Message(dir, uid); err == nil {
				_ = m.SetOneFlag(maildir.FlagSeen, i%2 == 0)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range n {
			_, _ = c.UIDs(dir)
			if m, err := c.Message(dir, uid); err == nil {
				_, _ = m.Flags()
				_, _ = m.MessageHeaders()
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range n {
			if m, err := c.Message(dir, uid); err == nil {
				_, _ = m.MessageHeaders()
				_, _ = m.Size()
			}
		}
	}()
	wg.Wait()

	// The message must still be there and readable.
	uids, err := c.UIDs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(uids) != 1 || uids[0] != uid {
		t.Fatalf("messages: %v, expected [%v]", uids, uid)
	}
	m, err := c.Message(dir, uid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.MessageHeaders(); err != nil {
		t.Fatal(err)
	}
}

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
