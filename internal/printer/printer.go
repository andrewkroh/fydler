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

package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"

	"github.com/andrewkroh/fydler/internal/analysis"
)

func JSON(diags []analysis.Diagnostic, w io.Writer) error {
	type Report struct {
		Diags []analysis.Diagnostic `json:"diagnostics"`
		Time  string                `json:"timestamp"`
		Args  []string              `json:"args,omitempty"`
	}

	r := Report{
		Diags: diags,
		Time:  time.Now().Format(time.RFC3339),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

func Text(diags []analysis.Diagnostic, w io.Writer) error {
	return text(diags, w, false)
}

func ColorText(diags []analysis.Diagnostic, w io.Writer) error {
	return text(diags, w, true)
}

func text(diags []analysis.Diagnostic, w io.Writer, wantColor bool) error {
	red := color.New(color.FgRed)
	bold := color.New(color.Bold)
	if !wantColor {
		red.DisableColor()
		bold.DisableColor()
	}

	var err error
	for _, d := range diags {
		if _, err = bold.Fprint(w, d.Pos); err != nil {
			return err
		}
		if _, err = red.Fprint(w, " ", d.Message); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(w, " (%s)\n", d.Category); err != nil {
			return err
		}

		for _, r := range d.Related {
			if _, err = bold.Fprintf(w, "  %s", r.Pos); err != nil {
				return err
			}
			if _, err = fmt.Fprintf(w, " %s (%s)\n", r.Message, d.Category); err != nil {
				return err
			}
		}
	}
	return nil
}
