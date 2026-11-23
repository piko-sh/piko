// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestDiagnosticPairsAndRepair(t *testing.T) {
	for index := range uint64(4) {
		savedCase := generate.New("source-diagnostics", 7, index)
		require.NoError(t, checkDiagnosticPair(t.Context(), nil, &savedCase), "index %d", index)

		savedCase.Files[savedCase.Expected.Diagnostic.BadPath] = savedCase.Files[savedCase.Expected.Diagnostic.GoodPath]
		assert.Error(t, checkDiagnosticPair(t.Context(), nil, &savedCase), "credited healthy control as diagnostic %d", index)
	}
}

func TestDiagnosticPairRequiresModel(t *testing.T) {
	requireProperty(t, checkDiagnosticPair(t.Context(), nil, &casefile.Case{}), "model")
}

func TestDiagnosticPairReportsMissingSource(t *testing.T) {
	savedCase := generate.New("source-diagnostics", 7, 0)
	delete(savedCase.Files, savedCase.Expected.Diagnostic.GoodPath)

	err := checkDiagnosticPair(t.Context(), nil, &savedCase)

	requireProperty(t, err, "missing-source")
	assert.Equal(t, savedCase.Expected.Diagnostic.GoodPath, resultOf(err).Source)
}
