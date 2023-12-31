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

package conflict

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Path             string
		Diags            []analysis.Diagnostic
		IgnoreTextFam    bool
		IgnoreKeywordFam bool
	}{
		{
			Path: "testdata/conflict.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/conflict.yml", Line: 2, Col: 3},
					Category: "conflict",
					Message:  "number has multiple data types (long, short)",
					Related: []analysis.RelatedInformation{
						{Pos: analysis.Pos{File: "testdata/conflict.yml", Line: 2, Col: 3}, Message: "long"},
						{Pos: analysis.Pos{File: "testdata/conflict.yml", Line: 4, Col: 3}, Message: "short"},
					},
				},
			},
		},
		{
			Path: "testdata/keyword_conflict.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 4, Col: 3},
					Category: "conflict",
					Message:  "id has multiple data types (constant_keyword, keyword, wildcard)",
					Related: []analysis.RelatedInformation{
						{Pos: analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 4, Col: 3}, Message: "constant_keyword"},
						{Pos: analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 2, Col: 3}, Message: "keyword"},
						{Pos: analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 6, Col: 3}, Message: "wildcard"},
					},
				},
			},
		},
		{
			Path:             "testdata/keyword_conflict.yml",
			IgnoreKeywordFam: true,
		},
		{
			Path: "testdata/text_conflict.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/text_conflict.yml", Line: 4, Col: 3},
					Category: "conflict",
					Message:  "abstract has multiple data types (match_only_text, text)",
					Related: []analysis.RelatedInformation{
						{Pos: analysis.Pos{File: "testdata/text_conflict.yml", Line: 4, Col: 3}, Message: "match_only_text"},
						{Pos: analysis.Pos{File: "testdata/text_conflict.yml", Line: 2, Col: 3}, Message: "text"},
					},
				},
			},
		},
		{
			Path:          "testdata/text_conflict.yml",
			IgnoreTextFam: true,
		},
		{
			Path: "testdata/ecs_conflict.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/ecs_conflict.yml", Line: 2, Col: 3},
					Category: "conflict",
					Message:  "message field declared as type text conflicts with the ECS data type match_only_text",
				},
			},
		},
		{
			Path:          "testdata/ecs_conflict.yml",
			IgnoreTextFam: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(filepath.Base(tc.Path), func(t *testing.T) {
			ignoreKeywordFamilyConflicts = tc.IgnoreKeywordFam
			ignoreTextFamilyConflicts = tc.IgnoreTextFam

			_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)
		})
	}
}
