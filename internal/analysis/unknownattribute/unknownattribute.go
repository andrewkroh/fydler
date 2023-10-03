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
	"errors"
	"fmt"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/yamledit"
)

var Analyzer = &analysis.Analyzer{
	Name:        "unknownattribute",
	Description: "Detect unknown field attributes.",
	CanFix:      true,
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Fields {
		for attrName := range f.AdditionalAttributes {
			if pass.Fix {
				fixed, err := fixUnknownAttribute(f, attrName, pass)
				if err != nil {
					return nil, err
				}
				if fixed {
					continue
				}
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

// fixUnknownAttribute removes attributes that are known to be useless. It must
// leave all other attributes in place because they may be typos of valid attributes.
func fixUnknownAttribute(field *fleetpkg.Field, attr string, pass *analysis.Pass) (fixed bool, err error) {
	if _, safe := safeToRemove[attr]; !safe {
		return false, nil
	}

	p, err := yaml.PathString(field.YAMLPath + "." + attr)
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
