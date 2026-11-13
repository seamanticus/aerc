package msgview

import (
	"net/url"

	"github.com/seamanticus/aerc/app"
	"github.com/seamanticus/aerc/commands"
	"github.com/seamanticus/aerc/lib/ui"
)

type CopyLink struct {
	Url *url.URL `opt:"url" action:"ParseUrl" complete:"CompleteUrl"`
}

func init() {
	commands.Register(CopyLink{})
}

func (CopyLink) Description() string {
	return "Copy the specified URL to the system clipboard."
}

func (CopyLink) Context() commands.CommandContext {
	return commands.MESSAGE_VIEWER
}

func (CopyLink) Aliases() []string {
	return []string{"copy-link"}
}

func (*CopyLink) CompleteUrl(arg string) []string {
	mv := app.SelectedTabContent().(*app.MessageViewer)
	if mv != nil {
		if p := mv.SelectedMessagePart(); p != nil {
			return commands.FilterList(p.Links, arg, nil)
		}
	}
	return nil
}

func (o *CopyLink) ParseUrl(arg string) error {
	u, err := url.Parse(arg)
	if err != nil {
		return err
	}
	o.Url = u
	return nil
}

func (o CopyLink) Execute(args []string) error {
	ui.PushClipboard(o.Url.String())
	return nil
}
