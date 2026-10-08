package maildir

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

func TestNewWorkerFSEvents(t *testing.T) {
	for name, newWorker := range map[string]func(*types.Worker) (types.Backend, error){
		"maildir":   NewWorker,
		"maildirpp": NewMaildirppWorker,
	} {
		t.Run(name, func(t *testing.T) {
			b, err := newWorker(types.NewWorker(name, nil))
			if err != nil {
				t.Skipf("cannot create file system watcher: %v", err)
			}
			// Run() selects on fsEvents. Sending to a nil channel
			// blocks forever, so file system changes would never be
			// processed.
			if b.(*Worker).fsEvents == nil {
				t.Fatal("fsEvents channel is not initialized")
			}
		})
	}
}
