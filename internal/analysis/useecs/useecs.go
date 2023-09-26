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

package useecs

import (
	"errors"
	"fmt"

	"github.com/andrewkroh/go-ecs"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "useecs",
	Description: "Detect fields that exist in the latest version of ECS, but are not using 'external: ecs'.",
	Run:         run,
}

var ignoreConstantKeyword bool

func init() {
	Analyzer.Flags.BoolVar(&ignoreConstantKeyword, "ignore-constant-keyword", false, "Ignore field definitions where ECS declares the type as 'keyword', but the definition uses the more optimized 'constant_keyword' type.")
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

		if ignoreConstantKeyword && ecsField.DataType == "keyword" && f.Type == "constant_keyword" {
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
