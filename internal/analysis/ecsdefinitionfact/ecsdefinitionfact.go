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

// Package ecsdefinitionfact provides a fact that gathers the external ECS
// definition for fields. It uses the ECS version fact to determine the
// version of ECS to use for the lookup.
package ecsdefinitionfact

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/andrewkroh/go-ecs"
	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/analysis/ecsversionfact"
)

var Analyzer = &analysis.Analyzer{
	Name:        "ecsdefinitionfact",
	Description: "Gathers the external ECS definition for fields.",
	Run:         run,
	Requires:    []*analysis.Analyzer{ecsversionfact.Analyzer},
}

type Fact struct {
	EnrichedFlat []*fleetpkg.Field // Field data enriched with external field data (type, description, pattern).
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Only log a Diagnostic once per directory.
	unknownECSVersion := map[string]struct{}{}
	ecsVersionsFact := pass.ResultOf[ecsversionfact.Analyzer].(*ecsversionfact.Fact)
	fact := &Fact{EnrichedFlat: make([]*fleetpkg.Field, 0, len(pass.Flat))}

	for _, f := range pass.Flat {
		if f.External != "ecs" {
			fact.EnrichedFlat = append(fact.EnrichedFlat, f)
			continue
		}

		// If the ecsVersion is not found, then the ecs.Lookup() will use data
		// from the latest ECS version. The ecsversionfact will have logged a
		// diagnostic about that problem.
		ecsVersion := ecsVersionsFact.ECSVersion(f.Path())

		dir := filepath.Dir(f.Path())
		ecsField, err := ecs.Lookup(f.Name, ecsVersion)
		if err != nil {
			switch {
			case errors.Is(err, ecs.ErrFieldNotFound):
				pass.Report(analysis.Diagnostic{
					Pos:      analysis.NewPos(f.FileMetadata),
					Category: pass.Analyzer.Name,
					Message:  fmt.Sprintf("%s is declared with 'external: ecs' but this field does not exist in ECS version %q", f.Name, ecsVersion),
				})
			case errors.Is(err, ecs.ErrVersionNotFound):
				if _, found := unknownECSVersion[dir]; !found {
					unknownECSVersion[dir] = struct{}{}
					pass.Report(analysis.Diagnostic{
						Pos:      analysis.NewPos(f.FileMetadata),
						Category: pass.Analyzer.Name,
						Message:  fmt.Sprintf("%s is declared with 'external: ecs' using ECS version %q, but this version is unknown this tool", f.Name, ecsVersion),
					})
				}
			case errors.Is(err, ecs.ErrInvalidVersion):
				if _, found := unknownECSVersion[dir]; !found {
					unknownECSVersion[dir] = struct{}{}
					pass.Report(analysis.Diagnostic{
						Pos:      analysis.NewPos(f.FileMetadata),
						Category: pass.Analyzer.Name,
						Message:  fmt.Sprintf("%s is declared with 'external: ecs' using ECS version %q, but that is an invalid version (%s)", f.Name, ecsVersion, err),
					})
				}
			default:
				return nil, fmt.Errorf("failed looking up ECS definition of %q from %s:%d:%d using version %q: %w",
					f.Name, f.Path(), f.Line(), f.Column(), ecsVersion, err)
			}

			continue
		}

		// Copy-on-write.
		{
			tmp := *f
			f = &tmp
		}
		f.Type = ecsField.DataType
		f.Pattern = ecsField.Pattern
		f.Description = ecsField.Description

		fact.EnrichedFlat = append(fact.EnrichedFlat, f)
	}

	return fact, nil
}
