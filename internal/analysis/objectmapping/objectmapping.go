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

// Package objectmapping performs analysis of fields to ensure they
// provide a specific mapping for all fields. Relates to
// https://github.com/elastic/package-spec/pull/628.
package objectmapping

import (
	"fmt"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "objectmapping",
	Description: "Detect fields that use an imprecise 'type: object' mapping.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// This cannot be implemented using pass.Flat because we need to use
	// f.Fields to filter invalid usages of 'type: object' on things that
	// should have been 'type: group'. This avoids several false positives.
	return nil, analysis.VisitFields(pass.Fields, func(f *fleetpkg.Field) error {
		if f.Type != "object" {
			return nil
		}

		// Only look at leaf fields.
		if len(f.Fields) > 0 {
			return nil
		}

		// Wildcard fields create dynamic_templates which are more precise mappings.
		if strings.Contains(f.Name, "*") {
			return nil
		}

		// Using 'type: object` and `object_type` creates a dynamic_templates entry
		// with 'path_match: {name}+".*"'.
		if f.ObjectType != "" {
			return nil
		}

		pass.Report(analysis.Diagnostic{
			Pos:      analysis.NewPos(f.FileMetadata),
			Category: pass.Analyzer.Name,
			Message:  fmt.Sprintf("%s uses an imprecise mapping, add specific mappings for subfields", f.Name),
		})

		return nil
	})
}
