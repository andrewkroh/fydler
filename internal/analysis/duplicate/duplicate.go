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

// Package duplicate detects duplicate field declarations within a directory.
package duplicate

import (
	"fmt"
	"path/filepath"

	"github.com/andrewkroh/go-fleetpkg"
	"golang.org/x/exp/maps"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "duplicate",
	Description: "Detect duplicate field declarations within a directory.",
	Run:         run,
}

func run(pass *analysis.Pass) (any, error) {
	seen := map[string][]*fleetpkg.Field{}
	var currentDir string

	flush := func() {
		for name, seenFields := range seen {
			if len(seenFields) < 2 {
				continue
			}

			field := seenFields[0]
			diag := analysis.Diagnostic{
				Pos:      analysis.NewPos(field.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s is declared %d times", name, len(seenFields)),
			}

			for _, f := range seenFields[1:] {
				diag.Related = append(diag.Related, analysis.RelatedInformation{
					Pos:     analysis.NewPos(f.FileMetadata),
					Message: "additional definition",
				})
			}

			pass.Report(diag)
		}
	}
	for _, f := range pass.Flat {
		// When the directory changes flush the duplicates.
		if dir := filepath.Dir(f.Path()); currentDir != dir {
			// Reset
			flush()
			maps.Clear(seen)
			currentDir = dir
		}

		seenFields := seen[f.Name]
		seenFields = append(seenFields, f)
		seen[f.Name] = seenFields
	}

	flush()
	return nil, nil
}
