package account

import (
	"errors"

	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/commands"
	"github.com/seamanticus/aerc/lib/ui"
)

type NextPrevResult struct{}

func init() {
	commands.Register(NextPrevResult{})
}

func (NextPrevResult) Description() string {
	return "Select the next or previous search result."
}

func (NextPrevResult) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (NextPrevResult) Aliases() []string {
	return []string{"next-result", "prev-result"}
}

func (NextPrevResult) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("no account selected")
	}
	if args[0] == "prev-result" {
		store := acct.Store()
		if store != nil {
			store.PrevResult()
		}
		ui.Invalidate()
	} else {
		store := acct.Store()
		if store != nil {
			store.NextResult()
		}
		ui.Invalidate()
	}
	return nil
}
