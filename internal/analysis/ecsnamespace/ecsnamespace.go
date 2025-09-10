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

// Package ecsnamespace provides an analyzer that detects fields being added to
// ECS managed namespaces. Fields should not be added to ECS managed namespaces.
// There are no exceptions.
//
// Information that does not have a place in ECS should be maintained within the
// data stream's field namespace which is canonically the data_stream.dataset
// value. That is, fields which are not in ECS should be prefixed by the
// dataset name (so a field in the audit data stream of the consul integration
// might be named consul.audit.accessor_id).
package ecsnamespace

import (
	"errors"
	"fmt"
	"strings"

	"github.com/andrewkroh/go-ecs"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "ecsnamespace",
	Description: "Detect fields being added to namespaces controlled by ECS.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	namespaces, err := ecsRootNamespaces()
	if err != nil {
		return nil, err
	}

	for _, f := range pass.Flat {
		// Ignore fields in ECS.
		if f.External == "ecs" {
			continue
		}

		// Ignore fields in ECS even if they are not using 'external: ecs'.
		ecsField, err := ecs.Lookup(f.Name, "")
		if err != nil && !errors.Is(err, ecs.ErrFieldNotFound) {
			// Should never happen.
			return nil, err
		}
		if ecsField != nil {
			continue
		}

		ns := namespace(f.Name)
		_, isECSManaged := namespaces[ns]
		if !isECSManaged {
			continue
		}

		pass.Report(analysis.Diagnostic{
			Pos:      analysis.NewPos(f.FileMetadata),
			Category: pass.Analyzer.Name,
			Message:  fmt.Sprintf("%s is defined in an ECS managed namespace, custom fields must use the dataset's namespace", f.Name),
		})
	}

	return nil, nil
}

func namespace(fieldName string) string {
	idx := strings.IndexByte(fieldName, '.')
	if idx == -1 {
		return ""
	}
	return fieldName[:idx]
}

func ecsRootNamespaces() (map[string]struct{}, error) {
	// Determine the ECS namespaces by using the latest version of ECS.
	fields, err := ecs.Fields("")
	if err != nil {
		return nil, err
	}

	namespaces := map[string]struct{}{}
	for name := range fields {
		ns := namespace(name)
		if ns == "" {
			continue
		}
		namespaces[ns] = struct{}{}
	}

	return namespaces, nil
}
