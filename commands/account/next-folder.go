package account

import (
	"errors"

	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/commands"
)

type NextPrevFolder struct {
	Unseen bool `opt:"-u" desc:"Cycle to the next or previous folder in the sidebar with unseen emails."`
	Offset int  `opt:"n" minus:"true" default:"1"`
}

func init() {
	commands.Register(NextPrevFolder{})
}

func (NextPrevFolder) Description() string {
	return "Cycle to the next or previous folder shown in the sidebar."
}

func (NextPrevFolder) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (NextPrevFolder) Aliases() []string {
	return []string{"next-folder", "prev-folder"}
}

func (np NextPrevFolder) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("no account selected")
	}
	if args[0] == "prev-folder" {
		acct.Directories().NextPrev(-np.Offset, np.Unseen)
	} else {
		acct.Directories().NextPrev(np.Offset, np.Unseen)
	}
	return nil
}
