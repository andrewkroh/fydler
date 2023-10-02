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

// Package fieldgroup performs analysis of field groups to
// ensure they contain a valid 'type'. Relates to
// github.com/elastic/package-spec/pull/629.
package fieldgroup

import (
	"fmt"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "fieldgroup",
	Description: "Detect fields groups with incorrect type.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Fields {
		err := analysis.VisitAll(f, func(f *fleetpkg.Field) error {
			// Only `type: group` and `type: nested` are allowed to have non-empty 'fields'.
			switch f.Type {
			case "group", "nested", "":
				return nil
			}

			if len(f.Fields) > 0 {
				pass.Report(analysis.Diagnostic{
					Pos:      analysis.NewPos(f.FileMetadata),
					Category: pass.Analyzer.Name,
					Message:  fmt.Sprintf("%s contains 'fields' and must be declared as 'type: group'", f.Name),
				})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
