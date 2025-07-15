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

// Package aliasfact provides a fact that resolves the type of alias target
// fields. It uses the ECS definition fact to ensure that any external ECS
// definitions are resolved. The fleetpkg.Field.Type is overwritten with the
// type of the target field. A diagnostic is reported if the target field does
// not exist in the same directory, and the unresolved alias field is included
// in the fact.
package aliasfact

import (
	"fmt"
	"path/filepath"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/analysis/ecsdefinitionfact"
)

var Analyzer = &analysis.Analyzer{
	Name: "aliasfact",
	Description: "Gathers the field type of the target field of an alias. " +
		"It reports a diagnostic if the target field does not resolve to a static field.",
	Run:      run,
	Requires: []*analysis.Analyzer{ecsdefinitionfact.Analyzer},
}

type Fact struct {
	ResolvedAliases []*fleetpkg.Field // Field data where the type is overwritten with the type of the target field.
}

func run(pass *analysis.Pass) (interface{}, error) {
	ecsDefinitionFact := pass.ResultOf[ecsdefinitionfact.Analyzer].(*ecsdefinitionfact.Fact)
	fact := &Fact{ResolvedAliases: make([]*fleetpkg.Field, 0, len(ecsDefinitionFact.EnrichedFlat))}

	for _, f := range ecsDefinitionFact.EnrichedFlat {
		if f.Type != "alias" {
			fact.ResolvedAliases = append(fact.ResolvedAliases, f)
			continue
		}
		dir := filepath.Dir(f.Path())

		var resolvedType string
		for _, aliased := range ecsDefinitionFact.EnrichedFlat {
			if f.AliasTargetPath == aliased.Name {
				aliasedDir := filepath.Dir(aliased.Path())
				if dir != aliasedDir {
					continue
				}

				resolvedType = aliased.Type
				break
			}
		}

		if resolvedType == "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s is declared as an alias, but the aliased field %s does not exist in the same directory", f.Name, f.AliasTargetPath),
			})

			// Put the unresolved alias into the list so that it can be considered by downstream analyzers.
			fact.ResolvedAliases = append(fact.ResolvedAliases, f)
			continue
		}

		// Copy-on-write.
		{
			tmp := *f
			f = &tmp
		}
		f.Type = resolvedType

		fact.ResolvedAliases = append(fact.ResolvedAliases, f)
	}

	return fact, nil
}
