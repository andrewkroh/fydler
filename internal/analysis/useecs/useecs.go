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

// Package useecs provides an analyzer that detects fields that exist in the
// latest version of ECS, but are not using 'external: ecs'.
// It can also fix the issue by replacing the field definition with a new one
// that uses 'external: ecs'.
package useecs

import (
	"errors"
	"fmt"

	"github.com/andrewkroh/go-ecs"
	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"
	yamlast "github.com/goccy/go-yaml/ast"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/yamledit"
)

var Analyzer = &analysis.Analyzer{
	Name:        "useecs",
	Description: "Detect fields that exist in the latest version of ECS, but are not using 'external: ecs'.",
	CanFix:      true,
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Flat {
		if f.External != "" {
			continue
		}

		ecsField, err := ecs.Lookup(f.Name, "")
		if err != nil {
			if errors.Is(err, ecs.ErrFieldNotFound) {
				continue
			}
			// Should never happen.
			return nil, err
		}

		fixed, err := fixWithExternalECS(f, ecsField, pass)
		if err != nil {
			return nil, err
		}
		if fixed {
			continue
		}

		message := fmt.Sprintf("%s exists in ECS, but the definition is not using 'external: ecs'.", f.Name)
		if f.Type != "" && ecsField.DataType != f.Type {
			message += fmt.Sprintf(" The ECS type is %s, but this uses %s", ecsField.DataType, f.Type)
		}

		pass.Report(analysis.Diagnostic{
			Pos:      analysis.NewPos(f.FileMetadata),
			Category: pass.Analyzer.Name,
			Message:  message,
		})
	}

	return nil, nil
}

// fixWithExternalECS replaces the field node with a new definition that uses
// 'external: ecs'. It will retain certain attributes that override indexing
// behavior of the field.
func fixWithExternalECS(field *fleetpkg.Field, ecsField *ecs.Field, pass *analysis.Pass) (fixed bool, err error) {
	if !pass.Fix {
		return false, nil
	}

	// An ECS keyword may be replaced with a constant_keyword.
	// Source: https://github.com/elastic/elastic-package/blob/cafa676c6ec7420e08023f9af98185b114879714/internal/fields/dependency_manager.go#L204
	overrideWithConstantKeyword := ecsField.DataType == "keyword" && field.Type == "constant_keyword"

	// The type must be the same in order to do the replacement safely.
	if field.Type != ecsField.DataType && !overrideWithConstantKeyword {
		return false, nil
	}

	// Get the old node.
	p, err := yaml.PathString(field.YAMLPath)
	if err != nil {
		return false, err
	}

	ast := pass.AST[field.Path()]

	n, err := p.FilterFile(ast.File)
	if err != nil {
		return false, fmt.Errorf("failed to get YAML node %q: %w", field.YAMLPath, err)
	}

	// This operates on pass.Flat where the field name is not the original
	// name from the YAML node. We need the original name to modify the YAML.
	var o fleetpkg.Field
	if err = yaml.NodeToValue(n, &o); err != nil {
		return false, fmt.Errorf("failed to read original node: %w", err)
	}

	newField := fleetpkg.Field{
		Name:     o.Name,
		External: "ecs",

		// constant_keyword fields should retain their type.
		Value: o.Value,

		// Keep these attributes because they are needed for TSDS.
		MetricType: o.MetricType,
		Dimension:  o.Dimension,

		// Keep special attributes that control indexing.
		DocValues: o.DocValues,
		Index:     o.Index,
		CopyTo:    o.CopyTo,
		Enabled:   o.Enabled,

		// Keep the unit type because ECS does not have this concept.
		Unit: o.Unit,
	}
	if overrideWithConstantKeyword {
		newField.Type = "constant_keyword"
	}

	replacement, err := yaml.ValueToNode(newField)
	if err != nil {
		return false, err
	}
	yamlast.Walk(yamledit.FieldAttributeOrder, replacement)

	if err = p.ReplaceWithNode(ast.File, replacement); err != nil {
		return false, fmt.Errorf("faield to replace node: %w", err)
	}

	ast.Modified = true
	return true, nil
}
