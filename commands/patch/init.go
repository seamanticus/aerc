package patch

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seamanticus/aerc/commands"
	"github.com/seamanticus/aerc/lib/pama"
)

type Init struct {
	Force bool   `opt:"-f" desc:"Overwrite any existing project."`
	Name  string `opt:"name" required:"false"`
}

func init() {
	register(Init{})
}

func (Init) Description() string {
	return "Create a new project."
}

func (Init) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Init) Aliases() []string {
	return []string{"init"}
}

func (i Init) Execute(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current directory: %w", err)
	}

	name := i.Name
	if name == "" {
		name = filepath.Base(cwd)
	}

	return pama.New().Init(name, cwd, i.Force)
}
