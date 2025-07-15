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

// Package ecsversionfact provides a fact that returns the ECS version
// associated with a fields.yml file. The ECS version is determined
// by looking for a build.yml file for the package containing the fields.
package ecsversionfact

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"
	"gopkg.in/yaml.v3"

	"github.com/andrewkroh/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "ecsversionfact",
	Description: "Gathers the ECS version associated with fields. " +
		"It reports a diagnostic if the ECS version has not been specified.",
	Run: run,
}

type Fact struct {
	dirToECSVersion map[string]string
}

// ECSVersion returns the ECS version associated to a given fields.yml file.
func (f *Fact) ECSVersion(path string) string {
	return f.dirToECSVersion[filepath.Dir(path)]
}

func run(pass *analysis.Pass) (interface{}, error) {
	dirToECSVersion := map[string]string{}
	notExist := map[string]struct{}{}

	for _, f := range pass.Flat {
		if f.External != "ecs" {
			continue
		}

		dir := filepath.Dir(f.Path())
		if _, found := dirToECSVersion[dir]; found {
			continue
		}

		if _, found := notExist[dir]; found {
			continue
		}

		ecsRef, err := lookupECSReference(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				pass.Report(analysis.Diagnostic{
					Pos:      analysis.NewPos(f.FileMetadata),
					Category: pass.Analyzer.Name,
					Message:  "missing ecs version reference because build.yml not found",
				})
				notExist[dir] = struct{}{}
				continue
			}
			return nil, fmt.Errorf("failed to read ecs version: %w", err)
		}

		if ecsRef == "" {
			notExist[dir] = struct{}{}
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  "missing ecs version reference in build.yml",
			})
			continue
		}

		dirToECSVersion[dir] = ecsRef
	}

	return &Fact{dirToECSVersion: dirToECSVersion}, nil
}

func lookupECSReference(dir string) (string, error) {
	f, err := openBuildManifest(dir)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var manifest fleetpkg.BuildManifest
	dec := yaml.NewDecoder(f)
	if err = dec.Decode(&manifest); err != nil {
		return "", fmt.Errorf("failed to unmarshal %s: %w", f.Name(), err)
	}

	// Strip prefix from git@v1.2.3.
	gitRef := manifest.Dependencies.ECS.Reference
	gitRef = strings.TrimPrefix(gitRef, "git@")
	return gitRef, nil
}

// searchPaths contains relative paths from a fields.yml file to
// a package build.yml file.
var searchPaths = []string{
	// Integration data stream fields.
	"../../../_dev/build/build.yml",
	// Input package fields.
	"../_dev/build/build.yml",
	// Transform fields.
	"../../../../_dev/build/build.yml",
}

func openBuildManifest(dir string) (*os.File, error) {
	for _, searchPath := range searchPaths {
		f, err := os.Open(filepath.Join(dir, searchPath))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}

		return f, nil
	}

	return nil, fs.ErrNotExist
}
