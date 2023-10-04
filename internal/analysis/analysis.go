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

package analysis

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strconv"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	"github.com/andrewkroh/fydler/internal/yamledit"
)

type Analyzer struct {
	Name        string
	Description string
	CanFix      bool
	Requires    []*Analyzer

	Flags flag.FlagSet

	Run func(*Pass) (interface{}, error)
}

type Pass struct {
	Analyzer *Analyzer

	Fix bool // Should the analyzer apply fixes to AST?

	// Field information.
	Fields []*fleetpkg.Field // Fields from every file.
	Flat   []*fleetpkg.Field // Flat view of all fields sorted by file and line number.

	// Map of file paths to the AST of that file. This is available when Fix is true.
	// Analyzers may add, modify, and delete map attributes, but they should not
	// add or remove entire field list entries (any operation that changes indices
	// in YAML paths would break other analyzers).
	AST map[string]*AST

	// ResultOf provides the inputs to this analysis pass, which are
	// the corresponding results of its prerequisite analyzers.
	// The map keys are the elements of Analysis.Required,
	// and the type of each corresponding value is the required
	// analysis's ResultType.
	ResultOf map[*Analyzer]interface{}

	Report func(Diagnostic)
}

type Pos struct {
	File string
	Line int
	Col  int
}

func NewPos(meta fleetpkg.FileMetadata) Pos {
	return Pos{
		File: meta.Path(),
		Line: meta.Line(),
		Col:  meta.Column(),
	}
}

func (p Pos) String() string {
	if p.Col == 0 {
		return p.File + ":" + strconv.Itoa(p.Line)
	}
	return p.File + ":" + strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Col)
}

func (p Pos) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

type Diagnostic struct {
	Pos      Pos
	Category string
	Message  string
	Related  []RelatedInformation `json:"Related,omitempty"`
}

type RelatedInformation struct {
	Pos     Pos
	Message string
}

type AST struct {
	File     *ast.File
	Modified bool // Modified tracks whether File has been modified.
}

type Printer func(diags []Diagnostic, w io.Writer)

// VisitFields can be used to iterate over non-flat fields. Use this when you
// need to analyze attributes of non-leaf fields.
func VisitFields(fields []*fleetpkg.Field, v func(*fleetpkg.Field) error) error {
	for i := range fields {
		if err := visitField(fields[i], v); err != nil {
			return err
		}
	}
	return nil
}

func visitField(f *fleetpkg.Field, v func(*fleetpkg.Field) error) error {
	if err := v(f); err != nil {
		return err
	}
	for _, child := range f.Fields {
		if err := visitField(&child, v); err != nil {
			return err
		}
	}
	return nil
}

// DeleteKey deletes the specified key from the AST associated with the given field.
// If pass.Fix is false, then this is a no-op.
func DeleteKey(field *fleetpkg.Field, key string, pass *Pass) (modified bool, err error) {
	if !pass.Fix {
		return false, nil
	}

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
