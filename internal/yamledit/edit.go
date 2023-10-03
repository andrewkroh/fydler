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
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

// DeleteNode deletes a node from a map.
func DeleteNode(f *ast.File, p *yaml.Path) error {
	n, err := p.FilterFile(f)
	if err != nil {
		return err
	}

	for _, d := range f.Docs {
		m := ast.Parent(d, n)
		if m == nil {
			continue
		}
		switch p := ast.Parent(d, m).(type) {
		case *ast.MappingNode:
			for i, e := range p.Values {
				if e == m {
					p.Values = append(p.Values[:i], p.Values[i+1:]...)
					break
				}
			}
		default:
			return fmt.Errorf("failed to get parent node: %w", err)
		}
	}
	return nil
}

// SetString replaces the node at the specified path with a StringNode.
func SetString(f *ast.File, p *yaml.Path, value string) error {
	_, err := p.FilterFile(f)
	if err != nil {
		if yaml.IsNotFoundNodeError(err) {
			// If the key does not exist, then try to add it.
			parent, key, err := cutPath(p)
			if err != nil {
				return err
			}

			return appendMapNode(f, parent, key, value)
		}
		return err
	}

	replacement, err := yaml.ValueToNode(value)
	if err != nil {
		return err
	}

	return p.ReplaceWithNode(f, replacement)
}

// appendMapNode appends a new key/value to an existing map.
func appendMapNode(f *ast.File, p *yaml.Path, key string, value any) error {
	n, err := p.FilterFile(f)
	if err != nil {
		return err
	}

	// Build new mapping value.
	newNode, err := yaml.ValueToNode(map[string]any{
		key: value,
	})
	if err != nil {
		return err
	}
	newValue := newNode.(*ast.MappingNode).Values[0]

	// For maps with a single key. Relates https://github.com/goccy/go-yaml/issues/310.
	switch v := n.(type) {
	case *ast.MappingValueNode:
		n = ast.Mapping(
			token.New(":", ":", n.GetToken().Position),
			false,
			v)
	}

	switch n := n.(type) {
	case *ast.MappingNode:
		// Match indent.
		newValue.AddColumn(n.GetToken().Position.IndentNum)
		n.Values = append(n.Values, newValue)
	default:
		return fmt.Errorf("node found at path %s is not a map (found %T)", p.String(), n)
	}

	return nil
}

// cutPath slices the YAML path around the last dot.
func cutPath(p *yaml.Path) (before *yaml.Path, after string, err error) {
	pathStr := p.String()

	idx := strings.LastIndex(pathStr, ".")
	if idx < 0 {
		return nil, "", fmt.Errorf("cannot get parent path of %q", pathStr)
	}

	beforeStr := pathStr[:idx]
	after = pathStr[idx+1:]

	before, err = yaml.PathString(beforeStr)
	if err != nil {
		return nil, "", err
	}

	return before, after, nil
}
