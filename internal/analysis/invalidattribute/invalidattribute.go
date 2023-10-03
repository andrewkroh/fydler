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

package invalidattribute

import (
	"fmt"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "invalidattribute",
	Description: "Detect invalid usages of field attributes.",
	CanFix:      true,
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Fields {
		// 'description' on field groups is never used by anything in Fleet.
		if f.Type == "group" && f.Description != "" {
			var fixed bool
			if pass.Fix {
				var err error
				fixed, err = analysis.DeleteKey(f, "description", pass)
				if err != nil {
					return nil, err
				}
			}

			if !fixed {
				pass.Report(analysis.Diagnostic{
					Pos:      analysis.NewPos(f.FileMetadata),
					Category: pass.Analyzer.Name,
					Message:  fmt.Sprintf("%s field group contains a 'description', but this is unused by Fleet and can be removed", f.Name),
				})
			}
		}

		// It is invalid to specify a 'type' when an external definition is used.
		if f.Type != "" && f.External != "" {
			var fixed bool
			if pass.Fix {
				var err error
				fixed, err = analysis.DeleteKey(f, "type", pass)
				if err != nil {
					return nil, err
				}
			}

			if !fixed {
				pass.Report(analysis.Diagnostic{
					Pos:      analysis.NewPos(f.FileMetadata),
					Category: pass.Analyzer.Name,
					Message:  fmt.Sprintf("%s use 'external: %s', therefore 'type' should not be specified", f.Name, f.External),
				})
			}
		}
	}
	return nil, nil
}
