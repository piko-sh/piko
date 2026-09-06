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

package i18n_adapters

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestFlatBufferProvider_Load_PluralFormParseFailures(t *testing.T) {
	t.Parallel()

	storedParts := []i18n_domain.TemplatePart{{Kind: i18n_domain.PartLiteral, Literal: "stored"}}

	testCases := []struct {
		name          string
		wantPartsText string
		wantWarning   string
		pluralForms   []string
		wantFormKinds [][]i18n_domain.PartKind
	}{
		{
			name:          "a failing first form keeps the stored parts and renders literally",
			pluralForms:   []string{"one ${broken", "${count} items"},
			wantPartsText: "stored",
			wantFormKinds: [][]i18n_domain.PartKind{
				{i18n_domain.PartLiteral},
				{i18n_domain.PartExpression, i18n_domain.PartLiteral},
			},
			wantWarning: "en-GB:items[0]: Unterminated expression",
		},
		{
			name:          "a failing later form renders literally and the first form becomes the entry parts",
			pluralForms:   []string{"one item", "${count"},
			wantPartsText: "one item",
			wantFormKinds: [][]i18n_domain.PartKind{
				{i18n_domain.PartLiteral},
				{i18n_domain.PartLiteral},
			},
			wantWarning: "en-GB:items[1]: Unterminated expression",
		},
		{
			name:          "forms that all parse replace the stored parts without a warning",
			pluralForms:   []string{"one item", "many items"},
			wantPartsText: "one item",
			wantFormKinds: [][]i18n_domain.PartKind{
				{i18n_domain.PartLiteral},
				{i18n_domain.PartLiteral},
			},
			wantWarning: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
			require.NoError(t, err)
			defer func() { _ = sandbox.Close() }()

			store := i18n_domain.NewStore("en-GB")
			store.AddLocale("en-GB", map[string]*i18n_domain.Entry{
				"items": {
					Template:         "unused",
					Parts:            storedParts,
					PluralForms:      tc.pluralForms,
					PluralFormsParts: nil,
					HasPlurals:       true,
				},
			})
			require.NoError(t, NewFlatBufferEmitter(sandbox).Emit(context.Background(), store, "en-GB", "i18n.bin"))

			ctx, logOutput := captureLogs()

			loadedStore, err := newFlatBufferProvider(sandbox, "i18n.bin").load(ctx)
			require.NoError(t, err)

			entry, found := loadedStore.Get("en-GB", "items")
			require.True(t, found)
			require.Len(t, entry.Parts, 1)
			assert.Equal(t, tc.wantPartsText, entry.Parts[0].Literal)
			require.Len(t, entry.PluralFormsParts, len(tc.wantFormKinds))
			for index, wantKinds := range tc.wantFormKinds {
				assert.Equal(t, wantKinds, partKinds(entry.PluralFormsParts[index]), "plural form %d", index)
			}
			for index, form := range tc.pluralForms {
				if strings.Contains(tc.wantWarning, fmt.Sprintf("[%d]", index)) {
					assert.Equal(t, form, entry.PluralFormsParts[index][0].Literal, "a failing form renders its own text")
				}
			}

			if tc.wantWarning == "" {
				assert.NotContains(t, logOutput.String(), "could not be parsed")
				return
			}
			assert.Contains(t, logOutput.String(), "could not be parsed")
			assert.Contains(t, logOutput.String(), tc.wantWarning)
		})
	}
}

func TestJSONProvider_Load_ReportsTemplateProblems(t *testing.T) {
	t.Parallel()

	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()
	require.NoError(t, sandbox.MkdirAll("i18n", 0o750))
	require.NoError(t, sandbox.WriteFile("i18n/en-GB.json", []byte(`{"ok": "Hello ${name}", "broken": "Hello ${name"}`), 0o600))

	ctx, logOutput := captureLogs()

	store, err := newJSONProvider(sandbox, "i18n").load(ctx, "en-GB")
	require.NoError(t, err)

	entry, found := store.Get("en-GB", "broken")
	require.True(t, found)
	require.Len(t, entry.Parts, 1)
	assert.Equal(t, "Hello ${name", entry.Parts[0].Literal)
	assert.Contains(t, logOutput.String(), "en-GB:broken: Unterminated expression")
	assert.NotContains(t, logOutput.String(), "en-GB:ok")
}

func captureLogs() (context.Context, *bytes.Buffer) {
	logOutput := new(bytes.Buffer)
	ctx := logger_domain.WithLogger(context.Background(),
		logger_domain.New(slog.New(slog.NewTextHandler(logOutput, nil)), "i18n-test"))
	return ctx, logOutput
}

func partKinds(parts []i18n_domain.TemplatePart) []i18n_domain.PartKind {
	kinds := make([]i18n_domain.PartKind, 0, len(parts))
	for _, part := range parts {
		kinds = append(kinds, part.Kind)
	}
	return kinds
}
