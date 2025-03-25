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

// Package dynamicfield provides an analyzer for detecting issues with
// wildcard fields meant to be dynamic mappings.
package dynamicfield

import (
	"fmt"
	"strings"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "dynamicfield",
	Description: "Detect issues with wildcard fields meant to be dynamic mappings.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Flat {
		if !strings.Contains(f.Name, "*") {
			continue
		}

		// This is always an error. Real field names should never contain an asterisk ('*').
		// Without 'object_type' fleet creates a static mapping for a field whose literal
		// name includes '*' (e.g. 'tags.*').
		if f.Type == "object" && f.ObjectType == "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message: fmt.Sprintf("%s field is meant to be a dynamic mapping, but is missing an 'object_type' "+
					"so it will never be a dynamic mapping", f.Name),
			})
			continue
		}

		if f.Type == "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message: fmt.Sprintf("%s field is meant to be a dynamic mapping, but does not specify a 'type' "+
					"so it will never be a dynamic mapping", f.Name),
			})
			continue
		}
	}
	return nil, nil
}
