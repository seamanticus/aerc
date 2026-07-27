//go:build notmuch
// +build notmuch

package notmuch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

func TestCriteria(t *testing.T) {
	criteria := types.SearchCriteria{
		Match: &types.SearchCriteriaPart{
			From: []string{"from1", "from2"},
			To:   []string{"to"},
		},
	}
	query := translate(&criteria)
	assert.Equal(t, "((from:from1) or (from:from2)) and ((to:to))", query)
}

func TestExclude(t *testing.T) {
	criteria := types.SearchCriteria{
		Match: &types.SearchCriteriaPart{
			From: []string{"from"},
		},
		Exclude: &types.SearchCriteriaPart{
			Invert: true,
			To:     []string{"to1", "to2"},
		},
	}
	query := translate(&criteria)
	assert.Equal(t, "((from:from)) and (not ((to:to1) or (to:to2)))", query)
}
