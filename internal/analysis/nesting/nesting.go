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

// Package nesting detects fields that are nested below a scalar type field.
package nesting

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/analysis/ecsdefinitionfact"
)

var Analyzer = &analysis.Analyzer{
	Name:        "nesting",
	Description: "Detect fields that are nested below a scalar type field.",
	Run:         run,
	Requires:    []*analysis.Analyzer{ecsdefinitionfact.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	ecsDefinitionFact := pass.ResultOf[ecsdefinitionfact.Analyzer].(*ecsdefinitionfact.Fact)

	// Build map of parent field name to child field.
	parentChildRelations := map[string][]*fleetpkg.Field{}
	for _, f := range ecsDefinitionFact.EnrichedFlat {
		if idx := strings.LastIndexByte(f.Name, '.'); idx != -1 {
			parentName := f.Name[:idx]

			slice := parentChildRelations[parentName]
			slice = append(slice, f)
			parentChildRelations[parentName] = slice
		}
	}

	for _, f := range ecsDefinitionFact.EnrichedFlat {
		// Skip non-scalar field types.
		switch f.Type {
		case "group", "object", "nested", "array":
			continue
		}

		// Check if this scalar field has any children.
		children, found := parentChildRelations[f.Name]
		if !found {
			continue
		}

		pass.Report(makeDiag(f, children))
	}

	return nil, nil
}

func makeDiag(parent *fleetpkg.Field, children []*fleetpkg.Field) analysis.Diagnostic {
	diag := analysis.Diagnostic{
		Pos:      analysis.NewPos(parent.FileMetadata),
		Category: "nesting",
		Message:  fmt.Sprintf("%s is defined as a scalar type (%s), but sub-fields were found", parent.Name, parent.Type),
		Related:  make([]analysis.RelatedInformation, 0, len(children)),
	}

	// Sort the children for determinism.
	slices.SortFunc(children, compareFieldByFileMetadata)

	for _, f := range children {
		diag.Related = append(diag.Related, analysis.RelatedInformation{
			Pos:     analysis.NewPos(f.FileMetadata),
			Message: f.Name + " is sub-field with type " + f.Type,
		})
	}
	return diag
}

func compareFieldByFileMetadata(a, b *fleetpkg.Field) int {
	if c := cmp.Compare(a.Path(), b.Path()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Line(), b.Line()); c != 0 {
		return c
	}
	return cmp.Compare(a.Column(), b.Column())
}
