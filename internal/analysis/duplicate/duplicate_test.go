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

package duplicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andrewkroh/fydler/internal/analysis"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func Test(t *testing.T) {
	_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, "testdata/fields.yml")
	if err != nil {
		t.Fatal(err)
	}

	require.Len(t, diags, 1)

	d := diags[0]
	assert.Equal(t, "duplicate", d.Category)
	assert.Equal(t, "message is declared 2 times", d.Message)
	assert.Equal(t, "testdata/fields.yml", d.Pos.File)
	assert.Equal(t, 2, d.Pos.Line)
	assert.Equal(t, 3, d.Pos.Col)
	assert.Len(t, d.Related, 1)
}
