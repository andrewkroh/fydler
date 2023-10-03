// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
