package maildir

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/emersion/go-maildir"

	"git.sr.ht/~rjarry/aerc/models"
)

func addTestMessage(t *testing.T, d maildir.Dir, flags ...maildir.Flag) models.UID {
	t.Helper()
	msg, w, err := d.Create(flags)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("Subject: test\r\n\r\nbody\r\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return models.UID(msg.Key())
}

// deleteAll calls c.DeleteAll and fails the test instead of hanging forever
// if it does not return.
func deleteAll(
	t *testing.T, c *Container, d maildir.Dir, uids []models.UID,
) ([]models.UID, error) {
	t.Helper()
	type result struct {
		deleted []models.UID
		err     error
	}
	done := make(chan result, 1)
	go func() {
		deleted, err := c.DeleteAll(d, uids)
		done <- result{deleted, err}
	}()
	select {
	case res := <-done:
		return res.deleted, res.err
	case <-time.After(5 * time.Second):
		t.Fatal("DeleteAll did not return (deadlock?)")
		return nil, nil
	}
}

func TestContainerSameKeyInTwoFolders(t *testing.T) {
	c, err := NewContainer(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	inbox := c.Store.Dir("INBOX")
	archive := c.Store.Dir("Archive")
	for _, d := range []maildir.Dir{inbox, archive} {
		if err := d.Init(); err != nil {
			t.Fatal(err)
		}
	}
	uid := addTestMessage(t, inbox)
	// Copy the file with the same name, as an external tool may do.
	src, err := inbox.MessageByKey(string(uid))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src.Filename())
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(string(archive), "cur", filepath.Base(src.Filename()))
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// Listing the other folder must not hijack the cached message.
	if _, err := c.UIDs(inbox); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UIDs(archive); err != nil {
		t.Fatal(err)
	}
	m, err := c.Message(inbox, uid)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := m.loadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Filename() != src.Filename() {
		t.Fatalf("INBOX message resolved to %q, expected %q",
			msg.Filename(), src.Filename())
	}
}

func TestContainerDeleteAll(t *testing.T) {
	for _, maildirpp := range []bool{false, true} {
		name := "maildir"
		if maildirpp {
			name = "maildirpp"
		}
		t.Run(name, func(t *testing.T) {
			c, err := NewContainer(t.TempDir(), maildirpp)
			if err != nil {
				t.Fatal(err)
			}
			dir := c.Store.Dir("INBOX")
			if err := dir.Init(); err != nil {
				t.Fatal(err)
			}
			uids := []models.UID{
				addTestMessage(t, dir),
				addTestMessage(t, dir),
				addTestMessage(t, dir),
			}

			// Message not in the cache yet.
			deleted, err := deleteAll(t, c, dir, uids[:1])
			if err != nil {
				t.Fatalf("uncached: %v", err)
			}
			if !slices.Equal(deleted, uids[:1]) {
				t.Fatalf("uncached: deleted %v, expected %v", deleted, uids[:1])
			}

			// Messages already in the cache, as when the worker has
			// opened the directory.
			if _, err := c.UIDs(dir); err != nil {
				t.Fatal(err)
			}
			deleted, err = deleteAll(t, c, dir, uids[1:])
			if err != nil {
				t.Fatalf("cached: %v", err)
			}
			if !slices.Equal(deleted, uids[1:]) {
				t.Fatalf("cached: deleted %v, expected %v", deleted, uids[1:])
			}

			// Deletion stops at the first error.
			kept := addTestMessage(t, dir)
			deleted, err = deleteAll(t, c, dir, []models.UID{"missing", kept})
			if err == nil {
				t.Fatal("error: expected an error for a missing message")
			}
			if len(deleted) != 0 {
				t.Fatalf("error: deleted %v, expected nothing", deleted)
			}

			// The container lock must have been released on all paths.
			if !c.TryLock() {
				t.Fatal("container lock is still held")
			}
			c.Unlock()

			left, err := c.UIDs(dir)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(left, []models.UID{kept}) {
				t.Fatalf("messages left: %v, expected %v", left, []models.UID{kept})
			}
		})
	}
}
