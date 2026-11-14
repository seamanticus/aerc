package app

import (
	"testing"

	"github.com/seamanticus/aerc/config"
	"github.com/seamanticus/aerc/lib"
	"github.com/seamanticus/aerc/lib/ui"
	"github.com/seamanticus/aerc/models"
	"github.com/seamanticus/aerc/worker/types"
)

// newTestAccount returns an AccountView whose INBOX has the given server
// Unseen count and holds one message (uid "1") with the given flags.
func newTestAccount(t *testing.T, unseen int, flags models.Flags) (*AccountView, *models.Directory) {
	t.Helper()
	// onMessage ends with UpdateStatus(), which looks up the selected tab
	// of the global aerc; an empty tab list means "no account selected".
	if aerc.tabs == nil {
		aerc.tabs = ui.NewTabs(nil)
	}
	worker := types.NewWorker("test", make(chan types.WorkerMessage, 10))
	dirlist := &DirectoryList{
		store:    lib.NewDirStore(),
		worker:   worker,
		acctConf: &config.AccountConfig{Name: "test"},
	}
	dir := &models.Directory{Name: "INBOX", Unseen: unseen}
	store := lib.NewMessageStore(worker, "INBOX",
		func() *config.UIConfig { return new(config.UIConfig) },
		nil, nil, nil, nil, nil, nil, nil)
	dirlist.store.SetMessageStore(dir, store)
	store.Update(&types.MessageInfo{
		Message: types.RespondTo(&types.FetchMessageHeaders{Directory: "INBOX"}),
		Info: &models.MessageInfo{
			Uid:       "1",
			Directory: "INBOX",
			Flags:     flags,
			Envelope:  &models.Envelope{Subject: "test"},
		},
	})
	acct := &AccountView{
		acct:    &config.AccountConfig{Name: "test"},
		dirlist: dirlist,
		worker:  worker,
	}
	return acct, dir
}

func TestOnMessageUnseenCount(t *testing.T) {
	tests := []struct {
		name        string
		storeFlags  models.Flags
		serverFlags models.Flags
		update      *types.MessageInfo
		want        int
	}{
		{
			// The store holds the placeholder \Seen of a cached header;
			// the server Unseen count already includes this message.
			name:        "flags fetch of cached header",
			storeFlags:  models.SeenFlag,
			serverFlags: 0,
			update: &types.MessageInfo{
				Message: types.RespondTo(&types.FetchMessageFlags{
					Directory: "INBOX",
					Uids:      []models.UID{"1"},
				}),
				ReplaceFlags: true,
			},
			want: 5,
		},
		{
			name:        "unilateral update marks unread",
			storeFlags:  models.SeenFlag,
			serverFlags: 0,
			update:      &types.MessageInfo{ReplaceFlags: true},
			want:        6,
		},
		{
			name:        "flag command response marks read",
			storeFlags:  0,
			serverFlags: models.SeenFlag,
			update: &types.MessageInfo{
				Message: types.RespondTo(&types.FlagMessages{
					Directory: "INBOX",
					Enable:    true,
					Flags:     models.SeenFlag,
					Uids:      []models.UID{"1"},
				}),
				ReplaceFlags: true,
			},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acct, dir := newTestAccount(t, 5, tt.storeFlags)
			tt.update.Info = &models.MessageInfo{
				Uid:       "1",
				Directory: "INBOX",
				Flags:     tt.serverFlags,
			}
			acct.onMessage(tt.update)
			if dir.Unseen != tt.want {
				t.Errorf("Unseen = %d, want %d", dir.Unseen, tt.want)
			}
		})
	}
}
