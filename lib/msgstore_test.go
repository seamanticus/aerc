package lib_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type mockWorker struct {
	w *types.Worker
}

func newMockWorker() *mockWorker {
	ch := make(chan types.WorkerMessage, 10)
	return &mockWorker{w: types.NewWorker("test", ch)}
}

func ui() *config.UIConfig {
	return new(config.UIConfig)
}

func TestMessageStore_UpdateFlags(t *testing.T) {
	m := newMockWorker()
	store := lib.NewMessageStore(m.w, "INBOX", ui, nil, nil, nil, nil, nil, nil, nil)
	uid := models.UID("1")

	// Initial message with SeenFlag
	store.Update(&types.MessageInfo{
		Info: &models.MessageInfo{
			Uid:       uid,
			Directory: "INBOX",
			Flags:     models.SeenFlag,
			Envelope:  &models.Envelope{Subject: "initial"},
		},
	})

	// check that it was added
	if _, ok := store.Messages[uid]; !ok {
		t.Fatal("Message was not added to store")
	}
	if !store.Messages[uid].Flags.Has(models.SeenFlag) {
		t.Fatal("SeenFlag was not set initially")
	}

	// Simulate a header update without ReplaceFlags
	store.Update(&types.MessageInfo{
		Info: &models.MessageInfo{
			Uid:       uid,
			Directory: "INBOX",
			Envelope:  &models.Envelope{Subject: "updated"},
			Flags:     0,
		},
		ReplaceFlags: false,
	})

	if !store.Messages[uid].Flags.Has(models.SeenFlag) {
		t.Error("Expected SeenFlag to be preserved after update without ReplaceFlags")
	}

	// Simulate a flag update with ReplaceFlags
	store.Update(&types.MessageInfo{
		Info: &models.MessageInfo{
			Uid:       uid,
			Directory: "INBOX",
			Flags:     0,
			Envelope:  &models.Envelope{Subject: "updated"},
		},
		ReplaceFlags: true,
	})

	if store.Messages[uid].Flags.Has(models.SeenFlag) {
		t.Error("Expected SeenFlag to be removed after update with ReplaceFlags")
	}
}
