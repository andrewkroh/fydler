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
	"errors"
	"fmt"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/yamledit"
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
				fixed, err = deleteKey(f, "description", pass)
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
				fixed, err = deleteKey(f, "type", pass)
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

func deleteKey(field *fleetpkg.Field, key string, pass *analysis.Pass) (fixed bool, err error) {
	p, err := yaml.PathString(field.YAMLPath + "." + key)
	if err != nil {
		return false, err
	}

	ast := pass.AST[field.Path()]

	if err := yamledit.DeleteNode(ast.File, p); err != nil {
		if !errors.Is(err, yaml.ErrNotFoundNode) {
			return true, nil
		}
		return false, err
	}

	ast.Modified = true
	return true, nil
}
