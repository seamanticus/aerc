package compose

import (
	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/commands"
)

type Encrypt struct{}

func init() {
	commands.Register(Encrypt{})
}

func (Encrypt) Description() string {
	return "Toggle encryption of the message to all recipients."
}

func (Encrypt) Context() commands.CommandContext {
	return commands.COMPOSE_EDIT | commands.COMPOSE_REVIEW
}

func (Encrypt) Aliases() []string {
	return []string{"encrypt"}
}

func (Encrypt) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	composer.SetEncrypt(!composer.Encrypt())
	return nil
}
