package yamledit

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldAttributeOrder(t *testing.T) {
	testCases := []struct {
		in  []string
		out []string
	}{
		{
			in: []string{
				"external",
				"name",
				"description",
				"Foo",
				"Bar",
			},
			out: []string{
				"name",
				"external",
				"description",
				"Bar",
				"Foo",
			},
		},
	}

	for i, tc := range testCases {
		tc := tc

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			slices.SortStableFunc(tc.in, FieldAttributeOrder.Compare)
			assert.Equal(t, tc.out, tc.in)
		})
	}
}
