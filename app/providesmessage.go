package app

import (
	"github.com/seamanticus/aerc/lib"
	"github.com/seamanticus/aerc/lib/ui"
	"github.com/seamanticus/aerc/models"
)

type PartInfo struct {
	Index []int
	Msg   *models.MessageInfo
	Part  *models.BodyStructure
	Links []string
}

type ProvidesMessage interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() (*models.MessageInfo, error)
	SelectedMessagePart() *PartInfo
}

type ProvidesMessages interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() (*models.MessageInfo, error)
	MarkedMessages() ([]models.UID, error)
}
