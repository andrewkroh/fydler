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

package main

import (
	"github.com/andrewkroh/fydler/internal/analysis/conflict"
	"github.com/andrewkroh/fydler/internal/analysis/duplicate"
	"github.com/andrewkroh/fydler/internal/analysis/dynamicfield"
	"github.com/andrewkroh/fydler/internal/analysis/fieldgroup"
	"github.com/andrewkroh/fydler/internal/analysis/invalidattribute"
	"github.com/andrewkroh/fydler/internal/analysis/missingtype"
	"github.com/andrewkroh/fydler/internal/analysis/nesting"
	"github.com/andrewkroh/fydler/internal/analysis/objectmapping"
	"github.com/andrewkroh/fydler/internal/analysis/unknownattribute"
	"github.com/andrewkroh/fydler/internal/analysis/useecs"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func main() {
	fydler.Main(
		conflict.Analyzer,
		duplicate.Analyzer,
		dynamicfield.Analyzer,
		fieldgroup.Analyzer,
		invalidattribute.Analyzer,
		missingtype.Analyzer,
		nesting.Analyzer,
		objectmapping.Analyzer,
		unknownattribute.Analyzer,
		useecs.Analyzer,
	)
}
