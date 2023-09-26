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

package useecs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Path                  string
		Diags                 []analysis.Diagnostic
		IgnoreConstantKeyword bool
	}{
		{
			Path: "testdata/fields.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/fields.yml", Line: 2, Col: 3},
					Category: "useecs",
					Message:  "event.dataset exists in ECS, but the definition is not using 'external: ecs'. The ECS type is keyword, but this uses constant_keyword",
				},
			},
		},
		{
			Path:                  "testdata/fields.yml",
			IgnoreConstantKeyword: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(filepath.Base(tc.Path), func(t *testing.T) {
			ignoreConstantKeyword = tc.IgnoreConstantKeyword

			_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)
		})
	}
}
