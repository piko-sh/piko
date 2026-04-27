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

package wasm_adapters

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

func TestValidateGenerateRequest_AcceptsEmptySources(t *testing.T) {
	t.Parallel()

	limits := generatorLimits{MaxFileCount: 1, MaxTotalBytes: 1, MaxFileBytes: 1}
	response := validateGenerateRequest(&wasm_dto.GenerateFromSourcesRequest{}, limits)
	assert.Nil(t, response, "empty sources should fall through to the no-files-found path")
}

func TestValidateGenerateRequest_RejectsTooManyFiles(t *testing.T) {
	t.Parallel()

	limits := generatorLimits{MaxFileCount: 2, MaxTotalBytes: 1024, MaxFileBytes: 1024}
	sources := map[string]string{
		"a.pk": "a", "b.pk": "b", "c.pk": "c",
	}
	response := validateGenerateRequest(&wasm_dto.GenerateFromSourcesRequest{Sources: sources}, limits)
	require.NotNil(t, response)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "exceeds limit")
}

func TestValidateGenerateRequest_RejectsOversizedFile(t *testing.T) {
	t.Parallel()

	limits := generatorLimits{MaxFileCount: 10, MaxTotalBytes: 1024, MaxFileBytes: 4}
	sources := map[string]string{"big.pk": "longer than four bytes"}
	response := validateGenerateRequest(&wasm_dto.GenerateFromSourcesRequest{Sources: sources}, limits)
	require.NotNil(t, response)
	assert.Contains(t, response.Error, "per-file limit")
}

func TestValidateGenerateRequest_RejectsAggregateOverflow(t *testing.T) {
	t.Parallel()

	limits := generatorLimits{MaxFileCount: 10, MaxTotalBytes: 8, MaxFileBytes: 1024}
	sources := map[string]string{"a.pk": "12345", "b.pk": "12345"}
	response := validateGenerateRequest(&wasm_dto.GenerateFromSourcesRequest{Sources: sources}, limits)
	require.NotNil(t, response)
	assert.Contains(t, response.Error, "aggregate source size")
}

func TestValidateGenerateRequest_AllowsZeroLimitsAsUnlimited(t *testing.T) {
	t.Parallel()

	limits := generatorLimits{MaxFileCount: 0, MaxTotalBytes: 0, MaxFileBytes: 0}
	sources := map[string]string{"a.pk": strings.Repeat("x", 1024*1024)}
	response := validateGenerateRequest(&wasm_dto.GenerateFromSourcesRequest{Sources: sources}, limits)
	assert.Nil(t, response, "zero on every limit means unlimited")
}

func TestGeneratorAdapter_DefaultLimits(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter()
	assert.Equal(t, defaultMaxGenerateSourceFiles, adapter.limits.MaxFileCount)
	assert.Equal(t, defaultMaxGenerateSourceBytes, adapter.limits.MaxTotalBytes)
	assert.Equal(t, defaultMaxGenerateFileBytes, adapter.limits.MaxFileBytes)
}

func TestGeneratorAdapter_WithGeneratorLimitsOverride(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter(WithGeneratorLimits(7, 8, 9))
	assert.Equal(t, 7, adapter.limits.MaxFileCount)
	assert.Equal(t, 8, adapter.limits.MaxTotalBytes)
	assert.Equal(t, 9, adapter.limits.MaxFileBytes)
}

func TestGeneratorAdapter_Generate_RejectsOversizeRequest(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter(WithGeneratorLimits(1, 1024, 1024))
	request := &wasm_dto.GenerateFromSourcesRequest{
		Sources: map[string]string{"a.pk": "x", "b.pk": "y"},
	}
	response, err := adapter.Generate(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "exceeds limit", "the validation gate must reject before stdlib lookup")
}

func TestGeneratorAdapter_Generate_RejectsNilRequest(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter()
	response, err := adapter.Generate(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "nil")
}

func TestConvertArtefacts_PrecedenceGeneratorWinsOverFSWriter(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter()
	fsWriter := NewInMemoryFSWriter()
	require.NoError(t, fsWriter.WriteFile(context.Background(), "shared/path.go", []byte("from-fs")))

	generatorArtefacts := []*generator_dto.GeneratedArtefact{{
		SuggestedPath: "shared/path.go",
		Content:       []byte("from-generator"),
	}}

	result := adapter.convertArtefacts(generatorArtefacts, fsWriter, nil)
	assert.Len(t, result, 1, "duplicate path must be deduplicated")
	assert.Equal(t, "from-generator", result[0].Content, "generator artefact wins on collision")
}

func TestConvertArtefacts_PrecedenceFSWriterWinsOverPKJS(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter()
	fsWriter := NewInMemoryFSWriter()
	require.NoError(t, fsWriter.WriteFile(context.Background(), "pk-js/pages/x.js", []byte("from-fs")))

	pkJSEmitter := NewInMemoryPKJSEmitter()
	require.NoError(t, pkJSEmitter.Put("pk-js/pages/x.js", "from-pkjs"))

	result := adapter.convertArtefacts(nil, fsWriter, pkJSEmitter)
	assert.Len(t, result, 1)
	assert.Equal(t, "from-fs", result[0].Content, "fsWriter precedes pkJSEmitter on duplicate paths")
}

func TestConvertArtefacts_MergesUniquePathsFromAllThreeSources(t *testing.T) {
	t.Parallel()

	adapter := NewGeneratorAdapter()
	fsWriter := NewInMemoryFSWriter()
	require.NoError(t, fsWriter.WriteFile(context.Background(), "register.go", []byte("reg")))

	pkJSEmitter := NewInMemoryPKJSEmitter()
	require.NoError(t, pkJSEmitter.Put("pk-js/components/c.js", "comp"))

	generatorArtefacts := []*generator_dto.GeneratedArtefact{{
		SuggestedPath: "pages/p.go",
		Content:       []byte("page"),
	}}

	result := adapter.convertArtefacts(generatorArtefacts, fsWriter, pkJSEmitter)
	require.Len(t, result, 3)

	paths := make(map[string]string, len(result))
	for _, artefact := range result {
		paths[artefact.Path] = artefact.Content
	}
	assert.Equal(t, "page", paths["pages/p.go"])
	assert.Equal(t, "reg", paths["register.go"])
	assert.Equal(t, "comp", paths["pk-js/components/c.js"])
}

func TestParsePositiveInt(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input   string
		want    int
		wantErr bool
	}{
		"positive":      {"42", 42, false},
		"zero rejected": {"0", 0, true},
		"negative":      {"-1", 0, true},
		"empty":         {"", 0, true},
		"non-digit":     {"abc", 0, true},
	}
	for name, scenario := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := parsePositiveInt(scenario.input)
			if scenario.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, scenario.want, got)
		})
	}
}

func TestLocationFromError_ParsesGoStyleLineColumn(t *testing.T) {
	t.Parallel()

	err := newPositionedError("file.pkc:12:7: parse error: unexpected EOF")
	loc := locationFromError(err, "components/foo.pkc")
	assert.Equal(t, "components/foo.pkc", loc.FilePath)
	assert.Equal(t, 12, loc.Line)
	assert.Equal(t, 7, loc.Column)
}

func TestLocationFromError_NoMatchYieldsBareLocation(t *testing.T) {
	t.Parallel()

	err := newPositionedError("opaque error with no position")
	loc := locationFromError(err, "components/foo.pkc")
	assert.Equal(t, "components/foo.pkc", loc.FilePath)
	assert.Zero(t, loc.Line)
	assert.Zero(t, loc.Column)
}

func newPositionedError(s string) error {
	return &fixedError{message: s}
}

type fixedError struct {
	message string
}

func (e *fixedError) Error() string {
	return e.message
}
