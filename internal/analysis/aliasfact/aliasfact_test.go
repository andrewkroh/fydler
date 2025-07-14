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

package aliasfact

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Path   string
		Diags  []analysis.Diagnostic
		Fields map[string]string
	}{
		{
			Path: "testdata/my_package/data_stream/alias/fields/alias.yml",
			Fields: map[string]string{
				"body":    "match_only_text",
				"message": "match_only_text",
			},
		},
		{
			Path: "testdata/my_package/data_stream/unresolved_alias/fields/unresolved_alias.yml",
			Fields: map[string]string{
				"body": "alias",
			},
			Diags: []analysis.Diagnostic{
				{
					Pos: analysis.Pos{
						File: "testdata/my_package/data_stream/unresolved_alias/fields/unresolved_alias.yml",
						Line: 2,
						Col:  3,
					},
					Category: "aliasfact",
					Message:  "body is declared as an alias, but the aliased field message does not exist in the same directory",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(filepath.Base(tc.Path), func(t *testing.T) {
			results, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)

			fact := results[Analyzer].(*Fact)
			require.Len(t, fact.ResolvedAliases, len(tc.Fields), "unexpected ResolvedAliases length")
			for _, f := range fact.ResolvedAliases {
				assert.Equal(t, tc.Fields[f.Name], f.Type)
			}
		})
	}
}
