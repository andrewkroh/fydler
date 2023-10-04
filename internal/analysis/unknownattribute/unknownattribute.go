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

package unknownattribute

import (
	"fmt"
	"slices"

	"golang.org/x/exp/maps"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "unknownattribute",
	Description: "Detect unknown field attributes.",
	CanFix:      true,
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Fields {
		// Determinism
		attrs := maps.Keys(f.AdditionalAttributes)
		slices.Sort(attrs)

		for _, attrName := range attrs {
			fixed, err := deleteUnknownAttribute(f, attrName, pass)
			if err != nil {
				return nil, err
			}
			if fixed {
				continue
			}

			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s contains an unknown attribute %q", f.Name, attrName),
			})
		}
	}
	return nil, nil
}

// safeToRemove is a list of keys that are safe to remove. The fields
// have no use, and their definitions are tolerated by elastic/package-spec
// but ignored.
var safeToRemove = map[string]bool{
	"default_field": true,
	"footnote":      true,
	"format":        true,
	"group":         true,
	"level":         true,
	"norms":         true,
	"title":         true,
}

// deleteUnknownAttribute removes the attribute if it is an attribute that is
// known to be unused. It must leave all other attributes in place because they
// may be typos of valid attributes.
func deleteUnknownAttribute(field *fleetpkg.Field, attr string, pass *analysis.Pass) (fixed bool, err error) {
	if _, safe := safeToRemove[attr]; !safe {
		return false, nil
	}

	return analysis.DeleteKey(field, attr, pass)
}
