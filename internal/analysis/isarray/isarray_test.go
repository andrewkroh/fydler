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

package isarray

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Name  string
		Path  string
		Diags []analysis.Diagnostic
	}{
		{
			Name: "noncompliant",
			Path: "testdata/noncompliant/my_package/data_stream/foo/fields/fields.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: filepath.Join("testdata/noncompliant/my_package/data_stream/foo", "sample_event.json"), Line: 3},
					Category: "isarray",
					Message:  `ECS field "event.category" is defined as an array, but a scalar value was found`,
				},
				{
					Pos:      analysis.Pos{File: filepath.Join("testdata/noncompliant/my_package/data_stream/foo", "sample_event.json"), Line: 6},
					Category: "isarray",
					Message:  `ECS field "host.name" is defined as a scalar, but an array value was found`,
				},
				{
					Pos:      analysis.Pos{File: filepath.Join("testdata/noncompliant/my_package/data_stream/foo", "sample_event.json"), Line: 9},
					Category: "isarray",
					Message:  `ECS field "related.ip" is defined as an array, but a scalar value was found`,
				},
				{
					Pos:      analysis.Pos{File: filepath.Join("testdata/noncompliant/my_package/data_stream/foo/_dev/test/pipeline", "test-sample-expected.json"), Line: 5},
					Category: "isarray",
					Message:  `ECS field "event.category" is defined as an array, but a scalar value was found`,
				},
				{
					Pos:      analysis.Pos{File: filepath.Join("testdata/noncompliant/my_package/data_stream/foo/_dev/test/pipeline", "test-sample-expected.json"), Line: 6},
					Category: "isarray",
					Message:  `ECS field "event.type" is defined as an array, but a scalar value was found`,
				},
				{
					Pos:      analysis.Pos{File: filepath.Join("testdata/noncompliant/my_package/data_stream/foo/_dev/test/pipeline", "test-sample-expected.json"), Line: 8},
					Category: "isarray",
					Message:  `ECS field "tags" is defined as an array, but a scalar value was found`,
				},
				{
					Pos:      analysis.Pos{File: filepath.Join("testdata/noncompliant/my_package/data_stream/foo/elasticsearch/ingest_pipeline", "default.yml"), Line: 7},
					Category: "isarray",
					Message:  `append processor targets ECS field "host.name" which does not have array normalization`,
				},
			},
		},
		{
			Name: "compliant",
			Path: "testdata/compliant/my_package/data_stream/foo/fields/fields.yml",
		},
		{
			Name: "no_sample_event",
			Path: "testdata/no_sample_event/my_package/data_stream/foo/fields/fields.yml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)
		})
	}
}
