package commands

import (
	"errors"
	"fmt"

	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/config"
	"go.rockorager.dev/vaxis"
)

type SendKeys struct {
	Keys string `opt:"..."`
}

func init() {
	Register(SendKeys{})
}

func (SendKeys) Description() string {
	return "Send keystrokes to the currently visible terminal."
}

func (SendKeys) Context() CommandContext {
	return GLOBAL
}

func (SendKeys) Aliases() []string {
	return []string{"send-keys"}
}

func (s SendKeys) Execute(args []string) error {
	tab, ok := app.SelectedTabContent().(app.HasTerminal)
	if !ok {
		return errors.New("there is no terminal here")
	}

	term := tab.Terminal()
	if term == nil {
		return errors.New("the terminal is not active")
	}

	keys2send, err := config.ParseKeyStrokes(s.Keys)
	if err != nil {
		return fmt.Errorf("unable to parse keystroke: %q: %w", s.Keys, err)
	}

	for _, key := range keys2send {
		ev := vaxis.Key{
			Keycode:   key.Key,
			Modifiers: key.Modifiers,
		}
		term.Event(ev)
	}

	term.Invalidate()

	return nil
}
