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

package binder_test

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/apitest"
	internalbinder "piko.sh/piko/internal/binder"
	"piko.sh/piko/wdk/binder"
)

func TestBinderFacadeAPI(t *testing.T) {

	surface := apitest.Surface{

		"ConverterFunc": binder.ConverterFunc(nil),

		"RegisterConverter": binder.RegisterConverter,

		"SetMaxFormFields": binder.SetMaxFormFields,

		"SetMaxFormPathLength": binder.SetMaxFormPathLength,

		"SetMaxFormValueLength": binder.SetMaxFormValueLength,

		"SetMaxFormPathDepth": binder.SetMaxFormPathDepth,

		"SetMaxSliceSize": binder.SetMaxSliceSize,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}

type facadeItem struct {
	Inner struct {
		Value string
	}
	Name string
}

type facadeForm struct {
	Name  string
	Items []facadeItem
	Age   int
}

type facadeCode string

type rejectingValidator struct{}

func (rejectingValidator) Struct(any) error {
	return errors.New("rejected")
}

func TestBindPopulatesDestination(t *testing.T) {
	var form facadeForm

	err := binder.Bind(t.Context(), &form, map[string][]string{
		"Name":          {"Alice"},
		"Age":           {"30"},
		"Items[0].Name": {"first"},
	})

	require.NoError(t, err)
	assert.Equal(t, "Alice", form.Name)
	assert.Equal(t, 30, form.Age)
	require.Len(t, form.Items, 1)
	assert.Equal(t, "first", form.Items[0].Name)
}

func TestBindMapPopulatesDestination(t *testing.T) {
	var form facadeForm

	err := binder.BindMap(t.Context(), &form, map[string]any{"Name": "Bob", "Age": 41})

	require.NoError(t, err)
	assert.Equal(t, "Bob", form.Name)
	assert.Equal(t, 41, form.Age)
}

func TestBindJSONPopulatesDestination(t *testing.T) {
	var form facadeForm

	err := binder.BindJSON(t.Context(), &form, []byte(`{"Name":"Carol","Items":[{"Name":"x"}]}`))

	require.NoError(t, err)
	assert.Equal(t, "Carol", form.Name)
	require.Len(t, form.Items, 1)
	assert.Equal(t, "x", form.Items[0].Name)
}

func TestBindJSONSizeLimitMatchesPublicSentinel(t *testing.T) {
	var destination struct{ Value int }
	oversized := make([]byte, internalbinder.DefaultMaxBindJSONBytes+1)

	err := binder.BindJSON(t.Context(), &destination, oversized)
	require.ErrorIs(t, err, binder.ErrBindJSONTooLarge)

	err = binder.BindJSON(t.Context(), &destination, []byte("["))
	require.Error(t, err)
	require.NotErrorIs(t, err, binder.ErrBindJSONTooLarge)
}

func TestPerCallOptionsOverrideGlobalLimits(t *testing.T) {
	testCases := []struct {
		source  map[string][]string
		name    string
		options []binder.Option
		wantErr bool
	}{
		{
			name:    "field count",
			source:  map[string][]string{"Name": {"a"}, "Age": {"1"}},
			options: []binder.Option{binder.WithMaxFieldCount(1)},
			wantErr: true,
		},
		{
			name:    "value length",
			source:  map[string][]string{"Name": {"abcdef"}},
			options: []binder.Option{binder.WithMaxValueLength(3)},
			wantErr: true,
		},
		{
			name:    "path length",
			source:  map[string][]string{"Items[0].Name": {"a"}},
			options: []binder.Option{binder.WithMaxPathLength(4)},
			wantErr: true,
		},
		{
			name:    "path depth",
			source:  map[string][]string{"Items[0].Inner.Value": {"a"}},
			options: []binder.Option{binder.WithMaxPathDepth(1)},
			wantErr: true,
		},
		{
			name:    "slice size",
			source:  map[string][]string{"Items[5].Name": {"a"}},
			options: []binder.Option{binder.WithMaxSliceSize(2)},
			wantErr: true,
		},
		{
			name:    "document scale limits",
			source:  map[string][]string{"Name": {"a"}},
			options: []binder.Option{binder.WithDocumentScaleLimits()},
			wantErr: false,
		},
		{
			name:    "unknown keys ignored",
			source:  map[string][]string{"Items[0].Missing": {"a"}},
			options: []binder.Option{binder.IgnoreUnknownKeys(true)},
			wantErr: false,
		},
		{
			name:    "unknown keys rejected",
			source:  map[string][]string{"Items[0].Missing": {"a"}},
			options: []binder.Option{binder.IgnoreUnknownKeys(false)},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var form facadeForm

			err := binder.Bind(t.Context(), &form, testCase.source, testCase.options...)

			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGlobalLimitSettersApplyToLaterBinds(t *testing.T) {
	testCases := []struct {
		apply   func()
		restore func()
		source  map[string][]string
		name    string
	}{
		{
			name:    "form fields",
			apply:   func() { binder.SetMaxFormFields(1) },
			restore: func() { binder.SetMaxFormFields(1_000) },
			source:  map[string][]string{"Name": {"a"}, "Age": {"1"}},
		},
		{
			name:    "path length",
			apply:   func() { binder.SetMaxFormPathLength(4) },
			restore: func() { binder.SetMaxFormPathLength(4_096) },
			source:  map[string][]string{"Items[0].Name": {"a"}},
		},
		{
			name:    "value length",
			apply:   func() { binder.SetMaxFormValueLength(3) },
			restore: func() { binder.SetMaxFormValueLength(65_536) },
			source:  map[string][]string{"Name": {"abcdef"}},
		},
		{
			name:    "path depth",
			apply:   func() { binder.SetMaxFormPathDepth(1) },
			restore: func() { binder.SetMaxFormPathDepth(32) },
			source:  map[string][]string{"Items[0].Inner.Value": {"a"}},
		},
		{
			name:    "slice size",
			apply:   func() { binder.SetMaxSliceSize(2) },
			restore: func() { binder.SetMaxSliceSize(1_000) },
			source:  map[string][]string{"Items[5].Name": {"a"}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var form facadeForm
			require.NoError(t, binder.Bind(t.Context(), &form, testCase.source))

			testCase.apply()
			t.Cleanup(testCase.restore)

			require.Error(t, binder.Bind(t.Context(), &form, testCase.source))
		})
	}
}

func TestSetIgnoreUnknownKeysChangesTheGlobalDefault(t *testing.T) {
	var form facadeForm
	source := map[string][]string{"Items[0].Missing": {"a"}}

	binder.SetIgnoreUnknownKeys(true)
	t.Cleanup(func() { binder.SetIgnoreUnknownKeys(false) })

	require.NoError(t, binder.Bind(t.Context(), &form, source))
	require.Error(t, binder.Bind(t.Context(), &form, source, binder.IgnoreUnknownKeys(false)))
}

func TestWithValidationRunsTheConfiguredValidator(t *testing.T) {
	var form facadeForm
	source := map[string][]string{"Name": {"a"}}

	binder.SetStructValidator(rejectingValidator{})
	t.Cleanup(func() { binder.SetStructValidator(nil) })

	require.NoError(t, binder.Bind(t.Context(), &form, source))

	err := binder.Bind(t.Context(), &form, source, binder.WithValidation(true))
	require.ErrorIs(t, err, binder.ErrValidationFailed)
	_, ok := errors.AsType[*binder.ValidationFailedError](err)
	assert.True(t, ok)
}

func TestRegisterConverterAppliesToBinds(t *testing.T) {
	binder.RegisterConverter(facadeCode(""), func(value string) (reflect.Value, error) {
		return reflect.ValueOf(facadeCode("code-" + value)), nil
	})

	var form struct{ Code facadeCode }
	require.NoError(t, binder.Bind(t.Context(), &form, map[string][]string{"Code": {"7"}}))

	assert.Equal(t, facadeCode("code-7"), form.Code)
}

func TestActionInputSourceSelectsTheNestedMap(t *testing.T) {
	nested := map[string]any{"Name": "a"}
	arguments := map[string]any{"input": nested}

	assert.Equal(t, nested, binder.ActionInputSource(arguments, "input"))
	assert.Equal(t, arguments, binder.ActionInputSource(arguments, "absent"))
	assert.Empty(t, binder.ActionInputSource(map[string]any{"input": "scalar"}, "input"))
}
