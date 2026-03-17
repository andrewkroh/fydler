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

// Package isarray detects ECS array normalization compliance issues
// by checking sample events, pipeline test outputs, and ingest
// pipeline append processors.
package isarray

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	ecs "github.com/andrewkroh/go-ecs"
	"gopkg.in/yaml.v3"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/analysis/ecsversionfact"
)

var Analyzer = &analysis.Analyzer{
	Name: "isarray",
	Description: "Detects ECS array normalization compliance issues in " +
		"sample events, pipeline test outputs, and ingest pipelines.",
	Requires: []*analysis.Analyzer{ecsversionfact.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	ecsVersionsFact := pass.ResultOf[ecsversionfact.Analyzer].(*ecsversionfact.Fact)

	// Discover unique data stream roots from field file paths.
	// The data stream root is the parent of the fields/ directory.
	type dsInfo struct {
		root       string
		ecsVersion string
	}
	seen := map[string]struct{}{}
	var dataStreams []dsInfo
	for _, f := range pass.Flat {
		fieldsDir := filepath.Dir(f.FilePath())
		dsRoot := filepath.Dir(fieldsDir)
		if _, ok := seen[dsRoot]; ok {
			continue
		}
		seen[dsRoot] = struct{}{}
		dataStreams = append(dataStreams, dsInfo{
			root:       dsRoot,
			ecsVersion: ecsVersionsFact.ECSVersion(f.FilePath()),
		})
	}

	// Sort for deterministic output.
	slices.SortFunc(dataStreams, func(a, b dsInfo) int {
		return strings.Compare(a.root, b.root)
	})

	for _, ds := range dataStreams {
		ecsFields, err := ecs.Fields(ds.ecsVersion)
		if err != nil {
			// Fall back to latest ECS version.
			ecsFields, err = ecs.Fields("")
			if err != nil {
				return nil, fmt.Errorf("failed to load ECS fields: %w", err)
			}
		}

		// Build set of ECS fields with array normalization.
		arrayFields := make(map[string]bool, len(ecsFields))
		for name, f := range ecsFields {
			if f.Array {
				arrayFields[name] = true
			}
		}

		// Allow error.message to be used as an array even though ECS
		// does not define it with array normalization. Appending to
		// error.message is a widespread convention in integrations.
		arrayFields["error.message"] = true

		checkSampleEvent(ds.root, ecsFields, arrayFields, pass)
		checkPipelineTests(ds.root, ecsFields, arrayFields, pass)
		checkIngestPipelines(ds.root, ecsFields, arrayFields, pass)
	}

	return nil, nil
}

// checkSampleEvent checks the sample_event.json for ECS array normalization
// compliance: fields that should be arrays but aren't, and fields that are
// arrays but shouldn't be.
func checkSampleEvent(dsRoot string, ecsFields map[string]*ecs.Field, arrayFields map[string]bool, pass *analysis.Pass) {
	path := filepath.Join(dsRoot, "sample_event.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:      analysis.Pos{File: path},
			Category: pass.Analyzer.Name,
			Message:  fmt.Sprintf("failed to read sample event: %v", err),
		})
		return
	}

	reported := map[string]bool{}
	lineTable := buildLineTable(data)
	dec := json.NewDecoder(bytes.NewReader(data))

	// Read opening '{'.
	t, err := dec.Token()
	if err != nil || t != json.Delim('{') {
		return
	}

	walkJSONObject(dec, "", lineTable, checkArrayNormalization(path, ecsFields, arrayFields, pass, reported))
}

// checkPipelineTests checks pipeline test expected outputs for ECS array
// normalization compliance.
func checkPipelineTests(dsRoot string, ecsFields map[string]*ecs.Field, arrayFields map[string]bool, pass *analysis.Pass) {
	pattern := filepath.Join(dsRoot, "_dev", "test", "pipeline", "test-*-expected.json")
	matches, _ := filepath.Glob(pattern)

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		reported := map[string]bool{}
		lineTable := buildLineTable(data)
		dec := json.NewDecoder(bytes.NewReader(data))

		// Read opening '{' of outer object.
		t, err := dec.Token()
		if err != nil || t != json.Delim('{') {
			continue
		}

		fn := checkArrayNormalization(path, ecsFields, arrayFields, pass, reported)

		// Find the "expected" key and walk each document in its array.
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				break
			}
			key, ok := kt.(string)
			if !ok {
				break
			}
			if key != "expected" {
				skipOneValue(dec)
				continue
			}

			// Read opening '[' of expected array.
			t, err := dec.Token()
			if err != nil {
				break
			}
			if t != json.Delim('[') {
				break
			}

			// Walk each document in the expected array.
			for dec.More() {
				t, err := dec.Token()
				if err != nil {
					break
				}
				if t != json.Delim('{') {
					skipAfterToken(dec, t)
					continue
				}
				walkJSONObject(dec, "", lineTable, fn)
			}
			break
		}
	}
}

// checkArrayNormalization returns a walkJSONObject visitor that reports
// ECS fields with incorrect array normalization: fields that should be
// arrays but aren't, and fields that are arrays but shouldn't be.
func checkArrayNormalization(file string, ecsFields map[string]*ecs.Field, arrayFields map[string]bool, pass *analysis.Pass, reported map[string]bool) func(string, bool, int) {
	return func(fieldPath string, isArray bool, line int) {
		if _, isECS := ecsFields[fieldPath]; !isECS {
			return
		}
		if reported[fieldPath] {
			return
		}

		shouldBeArray := arrayFields[fieldPath]
		switch {
		case shouldBeArray && !isArray:
			reported[fieldPath] = true
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.Pos{File: file, Line: line},
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("ECS field %q is defined as an array, but a scalar value was found", fieldPath),
			})
		case !shouldBeArray && isArray:
			reported[fieldPath] = true
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.Pos{File: file, Line: line},
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("ECS field %q is defined as a scalar, but an array value was found", fieldPath),
			})
		}
	}
}

// walkJSONObject walks a JSON object after '{' has been consumed, calling fn
// for each key-value pair with the dotted field path, whether the value is a
// JSON array, and the line number. Keys are visited in sorted order for
// deterministic output.
func walkJSONObject(dec *json.Decoder, prefix string, lineTable []int, fn func(path string, isArray bool, line int)) {
	// Collect all key-value entries first to sort by key.
	type entry struct {
		path    string
		isArray bool
		line    int
		isObj   bool
	}
	var entries []entry

	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			break
		}
		key, ok := kt.(string)
		if !ok {
			break
		}

		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		line := offsetToLine(lineTable, int(dec.InputOffset()))

		vt, err := dec.Token()
		if err != nil {
			break
		}

		if d, ok := vt.(json.Delim); ok {
			switch d {
			case '{':
				entries = append(entries, entry{path: path, line: line, isObj: true})
				walkJSONObject(dec, path, lineTable, fn)
			case '[':
				entries = append(entries, entry{path: path, isArray: true, line: line})
				skipArray(dec)
			}
		} else {
			entries = append(entries, entry{path: path, line: line})
		}
	}
	// Read closing '}'.
	dec.Token() //nolint:errcheck

	// Sort entries by path for deterministic output.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})
	for _, e := range entries {
		fn(e.path, e.isArray, e.line)
	}
}

// --- JSON token helpers ---

func skipOneValue(dec *json.Decoder) {
	t, err := dec.Token()
	if err != nil {
		return
	}
	skipAfterToken(dec, t)
}

func skipAfterToken(dec *json.Decoder, t json.Token) {
	if d, ok := t.(json.Delim); ok {
		switch d {
		case '{':
			skipObject(dec)
		case '[':
			skipArray(dec)
		}
	}
}

func skipObject(dec *json.Decoder) {
	for dec.More() {
		dec.Token() //nolint:errcheck // key
		skipOneValue(dec)
	}
	dec.Token() //nolint:errcheck // '}'
}

func skipArray(dec *json.Decoder) {
	for dec.More() {
		skipOneValue(dec)
	}
	dec.Token() //nolint:errcheck // ']'
}

// --- Line number helpers ---

// buildLineTable returns the byte offset of the start of each line (1-indexed).
func buildLineTable(data []byte) []int {
	table := []int{0} // line 1 starts at offset 0
	for i, b := range data {
		if b == '\n' {
			table = append(table, i+1)
		}
	}
	return table
}

// offsetToLine returns the 1-indexed line number for a byte offset.
func offsetToLine(lineTable []int, offset int) int {
	lo, hi := 0, len(lineTable)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if lineTable[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}

// --- Ingest pipeline checks (YAML with yaml.Node for line numbers) ---

// checkIngestPipelines checks ingest pipeline YAML files for append
// processors that target ECS fields without array normalization.
func checkIngestPipelines(dsRoot string, ecsFields map[string]*ecs.Field, arrayFields map[string]bool, pass *analysis.Pass) {
	pattern := filepath.Join(dsRoot, "elasticsearch", "ingest_pipeline", "*.yml")
	matches, _ := filepath.Glob(pattern)

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var doc yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			continue
		}

		if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
			continue
		}

		root := doc.Content[0]
		if root.Kind != yaml.MappingNode {
			continue
		}

		reported := map[string]bool{}
		for i := 0; i < len(root.Content)-1; i += 2 {
			key := root.Content[i]
			value := root.Content[i+1]

			if (key.Value == "processors" || key.Value == "on_failure") && value.Kind == yaml.SequenceNode {
				checkProcessorNodes(path, value, ecsFields, arrayFields, pass, reported)
			}
		}
	}
}

// checkProcessorNodes inspects a YAML sequence of processors for append
// operations targeting non-array ECS fields.
func checkProcessorNodes(file string, seq *yaml.Node, ecsFields map[string]*ecs.Field, arrayFields map[string]bool, pass *analysis.Pass, reported map[string]bool) {
	for _, procNode := range seq.Content {
		if procNode.Kind != yaml.MappingNode {
			continue
		}

		for i := 0; i < len(procNode.Content)-1; i += 2 {
			procTypeKey := procNode.Content[i]
			procConfig := procNode.Content[i+1]

			if procConfig.Kind != yaml.MappingNode {
				continue
			}

			if procTypeKey.Value == "append" {
				checkAppendNode(file, procTypeKey, procConfig, ecsFields, arrayFields, pass, reported)
			}

			// Check foreach processor's inner processor.
			if procTypeKey.Value == "foreach" {
				for j := 0; j < len(procConfig.Content)-1; j += 2 {
					cfgKey := procConfig.Content[j]
					cfgVal := procConfig.Content[j+1]

					if cfgKey.Value == "processor" && cfgVal.Kind == yaml.MappingNode {
						fakeSeq := &yaml.Node{
							Kind:    yaml.SequenceNode,
							Content: []*yaml.Node{cfgVal},
						}
						checkProcessorNodes(file, fakeSeq, ecsFields, arrayFields, pass, reported)
					}
				}
			}

			// Recursively check on_failure blocks.
			for j := 0; j < len(procConfig.Content)-1; j += 2 {
				cfgKey := procConfig.Content[j]
				cfgVal := procConfig.Content[j+1]

				if cfgKey.Value == "on_failure" && cfgVal.Kind == yaml.SequenceNode {
					checkProcessorNodes(file, cfgVal, ecsFields, arrayFields, pass, reported)
				}
			}
		}
	}
}

func checkAppendNode(file string, procTypeKey *yaml.Node, config *yaml.Node, ecsFields map[string]*ecs.Field, arrayFields map[string]bool, pass *analysis.Pass, reported map[string]bool) {
	var fieldName string
	var fieldLine int

	for i := 0; i < len(config.Content)-1; i += 2 {
		key := config.Content[i]
		val := config.Content[i+1]

		if key.Value == "field" && val.Kind == yaml.ScalarNode {
			fieldName = val.Value
			fieldLine = procTypeKey.Line
			break
		}
	}

	if fieldName == "" || strings.Contains(fieldName, "{{") {
		return
	}

	if _, exists := ecsFields[fieldName]; !exists {
		return // Not an ECS field.
	}
	if arrayFields[fieldName] {
		return // Field allows array normalization.
	}

	if reported[fieldName] {
		return
	}
	reported[fieldName] = true

	pass.Report(analysis.Diagnostic{
		Pos:      analysis.Pos{File: file, Line: fieldLine},
		Category: pass.Analyzer.Name,
		Message:  fmt.Sprintf("append processor targets ECS field %q which does not have array normalization", fieldName),
	})
}
