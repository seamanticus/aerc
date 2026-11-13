package account

import (
	"errors"

	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/commands"
	"github.com/seamanticus/aerc/lib/sort"
	"github.com/seamanticus/aerc/lib/state"
	"github.com/seamanticus/aerc/worker/types"
)

type Sort struct {
	Unused struct{} `opt:"-"`
	// these fields are only used for completion
	Reverse  bool     `opt:"-r" desc:"Sort in the reverse order."`
	Criteria []string `opt:"criteria" complete:"CompleteCriteria" desc:"Sort criterion."`
}

func init() {
	commands.Register(Sort{})
}

func (Sort) Description() string {
	return "Sort the message list by the given criteria."
}

func (Sort) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (Sort) Aliases() []string {
	return []string{"sort"}
}

var supportedCriteria = []string{
	"arrival",
	"cc",
	"date",
	"from",
	"read",
	"size",
	"subject",
	"to",
	"flagged",
}

func (*Sort) CompleteCriteria(arg string) []string {
	return commands.FilterList(supportedCriteria, arg, nil)
}

func (Sort) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("no account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("messages still loading")
	}

	if c := store.Capabilities(); c != nil {
		if !c.Sort {
			return errors.New("sorting is not available for this backend")
		}
	}

	var err error
	var sortCriteria []*types.SortCriterion
	if len(args[1:]) == 0 {
		sortCriteria = acct.GetSortCriteria()
	} else {
		sortCriteria, err = sort.GetSortCriteria(args[1:])
		if err != nil {
			return err
		}
	}

	acct.SetStatus(state.Sorting(true))
	store.Sort(sortCriteria, func(msg types.WorkerMessage) {
		if _, ok := msg.(*types.Done); ok {
			acct.SetStatus(state.Sorting(false))
		}
	})
	return nil
}
