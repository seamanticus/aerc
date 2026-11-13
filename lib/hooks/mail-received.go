package hooks

import (
	"fmt"

	"github.com/seamanticus/aerc/config"
	"github.com/seamanticus/aerc/models"
)

type MailReceived struct {
	Account string
	Backend string
	Folder  string
	Role    string
	MsgInfo *models.MessageInfo
}

func (m *MailReceived) Cmd() string {
	return config.Hooks().MailReceived
}

func (m *MailReceived) Env() []string {
	from := m.MsgInfo.Envelope.From[0]
	return []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_ACCOUNT_BACKEND=%s", m.Backend),
		fmt.Sprintf("AERC_FOLDER=%s", m.Folder),
		fmt.Sprintf("AERC_FROM_NAME=%s", from.Name),
		fmt.Sprintf("AERC_FROM_ADDRESS=%s", from.Address),
		fmt.Sprintf("AERC_SUBJECT=%s", m.MsgInfo.Envelope.Subject),
		fmt.Sprintf("AERC_MESSAGE_ID=%s", m.MsgInfo.Envelope.MessageId),
		fmt.Sprintf("AERC_FOLDER_ROLE=%s", m.Role),
	}
}
