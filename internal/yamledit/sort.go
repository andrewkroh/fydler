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

package yamledit

import (
	"cmp"
	"slices"

	"github.com/goccy/go-yaml/ast"
)

// FieldAttributeOrder is an ast.Visitor that sorts maps keys for field definitions.
var FieldAttributeOrder = mapKeyOrder{
	"name":        -6,
	"type":        -5,
	"external":    -4,
	"value":       -3,
	"description": -2,
}

type mapKeyOrder map[string]int

func (o mapKeyOrder) Compare(a, b string) int {
	ai, hasA := o[a]
	bi, hasB := o[b]

	if hasA && hasB {
		return ai - bi
	} else if hasA {
		return ai
	} else if hasB {
		return -1 * bi
	}
	return cmp.Compare(a, b)
}

func (o mapKeyOrder) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.MappingNode:
		slices.SortFunc(n.Values, func(a, b *ast.MappingValueNode) int {
			return o.Compare(a.Key.String(), b.Key.String())
		})
	}
	return o
}
