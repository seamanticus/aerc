package lib_test

import (
	"reflect"
	"testing"

	"github.com/seamanticus/aerc/lib"
	"github.com/seamanticus/aerc/models"
)

func TestDirStore_List(t *testing.T) {
	dirs := []string{"a/c", "x", "a/b", "d"}
	dirstore := lib.NewDirStore()
	for _, d := range dirs {
		dirstore.SetMessageStore(&models.Directory{Name: d}, nil)
	}
	for range 10 {
		if !reflect.DeepEqual(dirstore.List(), dirs) {
			t.Errorf("order does not match")
			return
		}
	}
}
