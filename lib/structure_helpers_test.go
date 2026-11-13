package lib_test

import (
	"testing"

	"github.com/seamanticus/aerc/lib"
	"github.com/seamanticus/aerc/models"
)

func TestLib_FindAllNonMultipart(t *testing.T) {
	testStructure := &models.BodyStructure{
		MIMEType: "multipart",
		Parts: []*models.BodyStructure{
			{},
			{
				MIMEType: "multipart",
				Parts: []*models.BodyStructure{
					{},
					{},
				},
			},
			{},
		},
	}

	expected := [][]int{
		{1},
		{2, 1},
		{2, 2},
		{3},
	}

	parts := lib.FindAllNonMultipart(testStructure, nil, nil)

	if len(expected) != len(parts) {
		t.Errorf("incorrect dimensions; expected: %v, got: %v", expected, parts)
	}

	for i := range parts {
		if !lib.EqualParts(expected[i], parts[i]) {
			t.Errorf("incorrect values; expected: %v, got: %v", expected[i], parts[i])
		}
	}
}
