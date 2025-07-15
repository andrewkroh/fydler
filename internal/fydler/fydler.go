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

// Package fydler provides the core functionality for the fydler tool.
// It provides the logic for running analyzers based on the provided CLI flags.
package fydler

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	runtimedebug "runtime/debug"
	"runtime/pprof"
	"slices"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml/parser"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/printer"
)

var (
	analyzersFilter  stringListFlag
	outputTypes      stringListFlag
	diagnosticFilter stringListFlag
	fixFindings      bool
	cpuprofile       string
)

//nolint:revive // This is a pseudo main function so allow exits.
func Main(analyzers ...*analysis.Analyzer) {
	slices.SortFunc(analyzers, compareAnalyzer)

	progname := filepath.Base(os.Args[0])
	log.SetFlags(0)
	log.SetPrefix(progname + ": ")

	parseFlags(analyzers)

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal("Failed to create CPU profile file: ", err)
		}
		defer f.Close()
		if err = pprof.StartCPUProfile(f); err != nil {
			log.Fatal("Failed to start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if len(flag.Args()) == 0 {
		log.Fatal("Must pass a list of fields.yml files (e.g. **/fields/*.yml)")
	}

	files := make([]string, len(flag.Args()))
	copy(files, flag.Args())

	_, diags, err := Run(analyzers, files...)
	if err != nil {
		log.Fatal(err)
	}

	if len(diagnosticFilter) > 0 {
		diags = slices.DeleteFunc(diags, func(diag analysis.Diagnostic) bool {
			return !diagnosticContains(diagnosticFilter, &diag)
		})
	}

	for _, output := range outputTypes {
		switch output {
		case "color-text":
			err = printer.ColorText(diags, os.Stdout)
		case "text":
			err = printer.Text(diags, os.Stdout)
		case "json":
			err = printer.JSON(diags, os.Stdout)
		case "markdown":
			err = printer.Markdown(diags, os.Stdout, analyzers, version())
		default:
			panic("invalid output type")
		}
		if err != nil {
			log.Fatal(err)
		}
	}
}

//nolint:revive // This is used by a pseudo main function so allow exits.
func parseFlags(analyzers []*analysis.Analyzer) {
	for _, a := range analyzers {
		prefix := a.Name + "."

		a.Flags.VisitAll(func(f *flag.Flag) {
			name := prefix + f.Name
			flag.Var(f.Value, name, f.Usage)
		})
	}

	flag.Var(&analyzersFilter, "a", "Analyzers to run. By default all analyzers are included.")
	flag.BoolVar(&fixFindings, "fix", false, "Run analyzers and write fixes to fields files. "+
		"This will only execute the analyzers that support automatic fixing.")
	flag.Var(&diagnosticFilter, "i", "Include only diagnostics with a path containing this value. "+
		"If specified more than once, then diagnostics that match any value are included.")
	flag.Var(&outputTypes, "set-output", "Output type to use. Allowed types are color-text, text, "+
		"markdown, and json. Defaults to color-text.")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "Write cpu profile to this file")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintln(out, "fydler [flags] fields_yml_file ...")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "fylder examines fields.yml files and reports issues that it finds,")
		fmt.Fprintln(out, "such as an unknown attribute, duplicate field definition, or")
		fmt.Fprintln(out, "conflicting type definition with another package.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "fydler is normally invoked using a shell glob pattern to match")
		fmt.Fprintln(out, "the fields.yml files of interest.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "  fydler packages/my_package/**/fields/*.yml")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "If you want fydler to consider all packages as context to the")
		fmt.Fprintln(out, "analyzers while only having interest in the results related to a")
		fmt.Fprintln(out, "particular path then you can use the include filter (-i).")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "  fydler -i '/my_package/' packages/**/fields/*.yml")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "The included analyzers are:")
		fmt.Fprintln(out, "")

		tw := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
		for _, a := range analyzers {
			var autoFix string
			if a.CanFix {
				autoFix = "(fix)"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", a.Name, a.Description, autoFix)
		}
		tw.Flush()
		fmt.Fprintln(out, "")

		fmt.Fprintln(out, "Version:", version())
		fmt.Fprintln(out, "")

		fmt.Fprintln(out, "Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	for _, output := range outputTypes {
		switch output {
		case "color-text", "text", "markdown", "json":
		default:
			log.Printf("invalid output type %q", output)
			os.Exit(1)
		}
	}
	if len(outputTypes) == 0 {
		outputTypes = []string{"color-text"}
	}

	// Split analyzer filters and validate the values.
	var tmp []string
	for _, a := range analyzersFilter {
		x := strings.FieldsFunc(a, func(r rune) bool {
			return unicode.IsSpace(r) || unicode.IsPunct(r)
		})
		tmp = append(tmp, x...)
	}
	analyzersFilter = tmp
nextFilter:
	for _, name := range analyzersFilter {
		for _, a := range analyzers {
			if a.Name == name {
				continue nextFilter
			}
		}
		log.Printf("invalid analyzer name %q", name)
		os.Exit(1)
	}
}

func Run(analyzers []*analysis.Analyzer, files ...string) (results map[*analysis.Analyzer]any, diags []analysis.Diagnostic, err error) {
	slices.Sort(files)

	// Honor the analyzers filter.
	if len(analyzersFilter) > 0 {
		analyzers = filterAnalyzers(analyzers, analyzersFilter)
	}

	if fixFindings {
		// Only run analyzers that can fix.
		analyzers = slices.DeleteFunc(analyzers, func(a *analysis.Analyzer) bool {
			return !a.CanFix
		})
	}

	analyzers, err = dependencyOrder(analyzers)
	if err != nil {
		return nil, nil, err
	}

	fields, err := fleetpkg.ReadFields(files...)
	if err != nil {
		return nil, nil, err
	}
	slices.SortFunc(fields, compareFieldByFileMetadata)

	flat, err := fleetpkg.FlattenFields(fields)
	if err != nil {
		return nil, nil, err
	}
	slices.SortFunc(flat, compareFieldByFileMetadata)

	pass := &analysis.Pass{
		Fields: toPointerSlice(fields),
		Flat:   toPointerSlice(flat),
		Report: func(d analysis.Diagnostic) {
			diags = append(diags, d)
		},
	}
	results = map[*analysis.Analyzer]any{}

	if fixFindings {
		pass.AST, err = loadASTs(fields)
		if err != nil {
			return nil, nil, err
		}
	}

	for _, a := range analyzers {
		pass.Analyzer = a
		pass.Fix = fixFindings
		pass.ResultOf = map[*analysis.Analyzer]any{}
		for _, required := range a.Requires {
			pass.ResultOf[required] = results[required]
		}

		result, err := a.Run(pass)
		if err != nil {
			return nil, nil, fmt.Errorf("failed running %s analyzer: %w", a.Name, err)
		}
		results[a] = result
	}

	if fixFindings {
		for path, ast := range pass.AST {
			if !ast.Modified {
				continue
			}
			if err = os.WriteFile(path, []byte(ast.File.String()), 0o644); err != nil {
				return nil, nil, err
			}
		}
	}

	return results, diags, nil
}

func loadASTs(fields []fleetpkg.Field) (map[string]*analysis.AST, error) {
	m := map[string]*analysis.AST{}
	for _, field := range fields {
		if _, found := m[field.Path()]; found {
			continue
		}

		f, err := parser.ParseFile(field.Path(), parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("failed loading AST for %s: %w", field.Path(), err)
		}

		m[field.Path()] = &analysis.AST{File: f}
	}
	return m, nil
}

func compareFieldByFileMetadata(a, b fleetpkg.Field) int {
	return compareFileMetadata(a.FileMetadata, b.FileMetadata)
}

func compareFileMetadata(a, b fleetpkg.FileMetadata) int {
	if c := cmp.Compare(a.Path(), b.Path()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Line(), b.Line()); c != 0 {
		return c
	}
	return cmp.Compare(a.Column(), b.Column())
}

func compareAnalyzer(a, b *analysis.Analyzer) int {
	return cmp.Compare(a.Name, b.Name)
}

func toPointerSlice[T any](in []T) []*T {
	out := make([]*T, len(in))
	for i := range in {
		out[i] = &in[i]
	}
	return out
}

func diagnosticContains(contains []string, diag *analysis.Diagnostic) bool {
	for _, c := range contains {
		if strings.Contains(diag.Pos.File, c) {
			return true
		}
		for _, related := range diag.Related {
			if strings.Contains(related.Pos.File, c) {
				return true
			}
		}
	}
	return false
}

func version() string {
	bi, ok := runtimedebug.ReadBuildInfo()
	if !ok {
		return "no build info"
	}

	var revision, modified string
	for _, bs := range bi.Settings {
		switch bs.Key {
		case "vcs.revision":
			revision = bs.Value
		case "vcs.modified":
			modified = bs.Value
		}
	}

	if revision == "" {
		return bi.Main.Version
	}

	switch modified {
	case "true":
		return fmt.Sprintln(bi.Main.Version, revision, "(modified)")
	case "false":
		return fmt.Sprintln(bi.Main.Version, revision)
	default:
		// This should never happen.
		return fmt.Sprintln(bi.Main.Version, revision, modified)
	}
}

// filterAnalyzers returns the intersection of two sets.
// It does not modify the analyzers slice.
func filterAnalyzers(analyzers []*analysis.Analyzer, selected []string) []*analysis.Analyzer {
	// Worst case is O(n^2).
	out := make([]*analysis.Analyzer, 0, len(selected))
	for _, a := range analyzers {
		for _, name := range selected {
			if a.Name == name {
				out = append(out, a)
				break
			}
		}
	}
	return out
}
