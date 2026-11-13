package worker

// the following workers are always enabled
import (
	_ "github.com/seamanticus/aerc/worker/imap"
	_ "github.com/seamanticus/aerc/worker/jmap"
	_ "github.com/seamanticus/aerc/worker/maildir"
	_ "github.com/seamanticus/aerc/worker/mbox"
)
