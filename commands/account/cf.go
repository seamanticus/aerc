package account

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/go-opt/v2"
	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/commands"
	"github.com/seamanticus/aerc/lib/state"
	"github.com/seamanticus/aerc/worker/types"
)

type ChangeFolder struct {
	Account string `opt:"-a" complete:"CompleteAccount" desc:"Change to specified account."`
	Folder  string `opt:"..." complete:"CompleteFolder" desc:"Folder name."`
}

func init() {
	commands.Register(ChangeFolder{})
}

func (ChangeFolder) Description() string {
	return "Change the folder shown in the message list."
}

func (ChangeFolder) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (ChangeFolder) Aliases() []string {
	return []string{"cf"}
}

func (c *ChangeFolder) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, nil)
}

func (c *ChangeFolder) CompleteFolder(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}
	retval := commands.FilterList(
		acct.Directories().List(), arg,
		commands.QuoteSpace,
	)
	return retval
}

func (c ChangeFolder) Execute([]string) error {
	var target string
	var acct *app.AccountView
	var err error

	args := opt.LexArgs(c.Folder)

	if c.Account != "" {
		acct, err = app.Account(c.Account)
		if err != nil {
			return err
		}
	} else {
		acct = app.SelectedAccount()
		if acct == nil {
			return errors.New("no account selected")
		}
	}

	if args.Count() == 0 {
		return errors.New("<folder> is required. Usage: cf [-a <account>] <folder>")
	}

	if args.Count() != 1 {
		return errors.New("unexpected argument(s). Usage: cf [-a <account>] <folder>")
	}
	target = args.Arg(0)

	finalize := func(msg types.WorkerMessage) {
		handleDirOpenResponse(acct, msg)
	}

	dirlist := acct.Directories()
	if dirlist == nil {
		return errors.New("no directory list found")
	}

	if target == "-" {
		dir := dirlist.Previous()
		if dir != "" {
			target = dir
		} else {
			return errors.New("no previous folder to return to")
		}
	}

	dirlist.Open(target, 0*time.Second, finalize, false)

	return nil
}

func handleDirOpenResponse(acct *app.AccountView, msg types.WorkerMessage) {
	// As we're waiting for the worker to report status we must run
	// the rest of the actions in this callback.
	switch msg := msg.(type) {
	case *types.Error:
		app.PushError(msg.Error.Error())
	case *types.Done:
		// reset store filtering if we switched folders
		store := acct.Store()
		if store != nil {
			store.ApplyClear()
			acct.SetStatus(state.SearchFilterClear())
		}
		// focus account tab
		acct.Select()
	}
}
