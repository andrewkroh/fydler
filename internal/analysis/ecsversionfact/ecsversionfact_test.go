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

package ecsversionfact

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Name       string
		Path       string
		Diags      []analysis.Diagnostic
		ECSVersion string
		Error      string
	}{
		{
			Name:       "my_package",
			Path:       "testdata/my_package/data_stream/foo/fields/fields.yml",
			ECSVersion: "v8.9.0",
		},
		{
			Name: "missing_build_yml",
			Path: "testdata/missing_build_yml/data_stream/foo/fields/fields.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/missing_build_yml/data_stream/foo/fields/fields.yml", Line: 4, Col: 3},
					Category: "ecsversionfact", Message: "missing ecs version reference because build.yml not found",
				},
			},
		},
		{
			Name:  "malformed_build_yml",
			Path:  "testdata/malformed_build_yml/data_stream/foo/fields/fields.yml",
			Error: "failed running ecsversionfact analyzer: failed to read ecs version: failed to unmarshal",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			results, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if tc.Error != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.Error)
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)

			fact := results[Analyzer].(*Fact)
			assert.Equal(t, tc.ECSVersion, fact.ECSVersion(tc.Path))
		})
	}
}
