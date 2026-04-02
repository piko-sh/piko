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

package premailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOptions_Values(t *testing.T) {
	opts := defaultOptions()

	assert.False(t, opts.KeepBangImportant)
	assert.False(t, opts.RemoveClasses)
	assert.False(t, opts.RemoveIDs)
	assert.False(t, opts.MakeLeftoverImportant)
	assert.True(t, opts.ExpandShorthands)
	assert.Nil(t, opts.LinkQueryParams)
	assert.Nil(t, opts.Theme)
	assert.Empty(t, opts.ExternalCSS)
}

func TestApplyOptions_NoOpts(t *testing.T) {
	opts := applyOptions()

	expected := defaultOptions()
	assert.Equal(t, expected, opts)
}

func TestApplyOptions_SingleOption(t *testing.T) {
	opts := applyOptions(WithKeepBangImportant(true))

	assert.True(t, opts.KeepBangImportant)
	assert.True(t, opts.ExpandShorthands, "other defaults should remain")
}

func TestApplyOptions_AllOptions(t *testing.T) {
	params := map[string]string{"utm_source": "test"}
	theme := map[string]string{"primary": "#000"}

	opts := applyOptions(
		WithKeepBangImportant(true),
		WithRemoveClasses(true),
		WithRemoveIDs(true),
		WithMakeLeftoverImportant(true),
		WithExpandShorthands(false),
		WithLinkQueryParams(params),
		WithTheme(theme),
		WithExternalCSS("body { color: red; }"),
	)

	assert.True(t, opts.KeepBangImportant)
	assert.True(t, opts.RemoveClasses)
	assert.True(t, opts.RemoveIDs)
	assert.True(t, opts.MakeLeftoverImportant)
	assert.False(t, opts.ExpandShorthands)
	assert.Equal(t, params, opts.LinkQueryParams)
	assert.Equal(t, theme, opts.Theme)
	assert.Equal(t, "body { color: red; }", opts.ExternalCSS)
}

func TestToFunctionalOptions_DefaultReturnsEmpty(t *testing.T) {
	opts := defaultOptions()
	functional := opts.ToFunctionalOptions()

	assert.Empty(t, functional)
}

func TestToFunctionalOptions_AllSet(t *testing.T) {
	opts := &Options{
		KeepBangImportant:     true,
		RemoveClasses:         true,
		RemoveIDs:             true,
		MakeLeftoverImportant: true,
		ExpandShorthands:      false,
		LinkQueryParams:       map[string]string{"k": "v"},
		Theme:                 map[string]string{"c": "#fff"},
		ExternalCSS:           "h1 { font-size: 2em; }",
	}

	functional := opts.ToFunctionalOptions()
	require.NotEmpty(t, functional)

	rebuilt := applyOptions(functional...)
	assert.Equal(t, opts.KeepBangImportant, rebuilt.KeepBangImportant)
	assert.Equal(t, opts.RemoveClasses, rebuilt.RemoveClasses)
	assert.Equal(t, opts.RemoveIDs, rebuilt.RemoveIDs)
	assert.Equal(t, opts.MakeLeftoverImportant, rebuilt.MakeLeftoverImportant)
	assert.Equal(t, opts.ExpandShorthands, rebuilt.ExpandShorthands)
	assert.Equal(t, opts.LinkQueryParams, rebuilt.LinkQueryParams)
	assert.Equal(t, opts.Theme, rebuilt.Theme)
	assert.Equal(t, opts.ExternalCSS, rebuilt.ExternalCSS)
}

func TestToFunctionalOptions_ExpandShorthandsFalse(t *testing.T) {
	opts := &Options{
		ExpandShorthands: false,
	}

	functional := opts.ToFunctionalOptions()
	assert.NotEmpty(t, functional, "should include ExpandShorthands(false) since default is true")
}

func TestToFunctionalOptions_PartialFields(t *testing.T) {
	opts := &Options{
		RemoveClasses:    true,
		ExpandShorthands: true,
	}

	functional := opts.ToFunctionalOptions()
	require.Len(t, functional, 1, "should only include RemoveClasses")

	rebuilt := applyOptions(functional...)
	assert.True(t, rebuilt.RemoveClasses)
	assert.True(t, rebuilt.ExpandShorthands, "default should remain")
}
