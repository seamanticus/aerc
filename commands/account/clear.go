package account

import (
	"errors"

	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/commands"
	"github.com/seamanticus/aerc/lib/state"
)

type Clear struct {
	Selected bool `opt:"-s" desc:"Select first message after clearing."`
}

func init() {
	commands.Register(Clear{})
}

func (Clear) Description() string {
	return "Clear the current search or filter criteria."
}

func (Clear) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (Clear) Aliases() []string {
	return []string{"clear"}
}

func (c Clear) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("no account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("cannot perform action: messages still loading")
	}

	if c.Selected {
		defer store.Select("")
	}
	store.ApplyClear()
	acct.SetStatus(state.SearchFilterClear())

	return nil
}
