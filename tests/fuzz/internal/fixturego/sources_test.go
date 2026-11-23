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

package fixturego

import (
	"errors"
	"go/ast"
	"go/token"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedRendersParseableAsset(t *testing.T) {
	source := Fixed("pkg/invoice/dto.go")
	assert.Contains(t, source, "package invoice")
}

func TestRenderAppliesEdit(t *testing.T) {
	source, err := Render("pkg/invoice/dto.go", func(_ *token.FileSet, file *ast.File) error {
		file.Name.Name = "renamed"
		return nil
	})
	require.NoError(t, err)
	assert.Contains(t, source, "package renamed")
}

func TestFixedPanicsOnMissingAsset(t *testing.T) {
	assert.Panics(t, func() { Fixed("missing.go") })
}

func TestRenderReturnsEditError(t *testing.T) {
	failure := errors.New("stale edit")
	_, err := Render("pkg/invoice/dto.go", func(*token.FileSet, *ast.File) error { return failure })
	require.ErrorIs(t, err, failure)
}

func TestRenderReturnsMissingAssetError(t *testing.T) {
	_, err := Render("missing.go", nil)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestSourceSHA256IsStableAndContentSensitive(t *testing.T) {
	assert.Equal(t, SourceSHA256(), SourceSHA256())
	assert.Len(t, SourceSHA256(), 64)
	first, err := sourceSHA256(fstest.MapFS{"a.go": {Data: []byte("package a")}})
	require.NoError(t, err)
	second, err := sourceSHA256(fstest.MapFS{"a.go": {Data: []byte("package b")}})
	require.NoError(t, err)
	assert.NotEqual(t, first, second)
}
