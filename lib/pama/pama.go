package pama

import (
	"fmt"

	"github.com/seamanticus/aerc/lib/pama/models"
	"github.com/seamanticus/aerc/lib/pama/revctrl"
	"github.com/seamanticus/aerc/lib/pama/store"
)

type (
	detectFn func(string) (string, string, error)
	rcFn     func(string, string) (models.RevisionController, error)
	storeFn  func() models.PersistentStorer
)

type PatchManager struct {
	detect detectFn
	rc     rcFn
	store  storeFn
}

func New() PatchManager {
	return PatchManager{
		detect: revctrl.Detect,
		rc:     revctrl.New,
		store:  store.Store,
	}
}

func FromFunc(d detectFn, r rcFn, s storeFn) PatchManager {
	return PatchManager{
		detect: d,
		rc:     r,
		store:  s,
	}
}

func storeErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("store error: %w", err)
}

func revErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("revision control error: %w", err)
}
