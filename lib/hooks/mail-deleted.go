package hooks

import (
	"fmt"

	"github.com/seamanticus/aerc/config"
)

type MailDeleted struct {
	Account string
	Backend string
	Folder  string
	Role    string
}

func (m *MailDeleted) Cmd() string {
	return config.Hooks().MailDeleted
}

func (m *MailDeleted) Env() []string {
	return []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_ACCOUNT_BACKEND=%s", m.Backend),
		fmt.Sprintf("AERC_FOLDER=%s", m.Folder),
		fmt.Sprintf("AERC_FOLDER_ROLE=%s", m.Role),
	}
}
