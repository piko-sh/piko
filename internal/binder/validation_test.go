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

package binder

import (
	"context"
	"errors"
	"strings"
	"testing"
)

var errInvalid = errors.New("stub validation failed")

type stubValidator struct {
	calls  int
	seen   any
	reject bool
}

func (v *stubValidator) Struct(s any) error {
	v.calls++
	v.seen = s
	if v.reject {
		return errInvalid
	}
	return nil
}

type validationTarget struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestBindValidation(t *testing.T) {
	testCases := []struct {
		name        string
		setValidate bool
		validate    bool
		validator   *stubValidator
		wantCalls   int
		wantErr     bool
	}{
		{
			name:        "validates when opted in",
			setValidate: true,
			validate:    true,
			validator:   &stubValidator{},
			wantCalls:   1,
			wantErr:     false,
		},
		{
			name:        "surfaces validation failure as an error",
			setValidate: true,
			validate:    true,
			validator:   &stubValidator{reject: true},
			wantCalls:   1,
			wantErr:     true,
		},
		{
			name:        "skips validation when not opted in",
			setValidate: false,
			validator:   &stubValidator{reject: true},
			wantCalls:   0,
			wantErr:     false,
		},
		{
			name:        "skips validation when explicitly disabled",
			setValidate: true,
			validate:    false,
			validator:   &stubValidator{reject: true},
			wantCalls:   0,
			wantErr:     false,
		},
		{
			name:        "opting in without a validator is a no-op",
			setValidate: true,
			validate:    true,
			validator:   nil,
			wantCalls:   0,
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewASTBinder()
			if tc.validator != nil {
				b.SetStructValidator(tc.validator)
			}

			opts := []Option{IgnoreUnknownKeys(true)}
			if tc.setValidate {
				opts = append(opts, WithValidation(tc.validate))
			}

			var dst validationTarget
			err := b.Bind(context.Background(), &dst, map[string][]string{"name": {"Morgan"}}, opts...)

			if tc.wantErr && err == nil {
				t.Fatal("expected a validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr && !errors.Is(err, errInvalid) {
				t.Fatalf("error should wrap the validator's failure, got %v", err)
			}
			if tc.validator != nil && tc.validator.calls != tc.wantCalls {
				t.Fatalf("validator calls = %d, want %d", tc.validator.calls, tc.wantCalls)
			}

			if dst.Name != "Morgan" {
				t.Fatalf("destination not bound: %+v", dst)
			}
		})
	}
}

func TestBindValidationSkippedWhenBindingFails(t *testing.T) {
	b := NewASTBinder()
	validator := &stubValidator{}
	b.SetStructValidator(validator)

	var dst validationTarget
	err := b.Bind(context.Background(), &dst,
		map[string][]string{"count": {"not-a-number"}},
		IgnoreUnknownKeys(true), WithValidation(true))

	if err == nil {
		t.Fatal("expected a binding error for the unconvertible value")
	}
	if validator.calls != 0 {
		t.Fatalf("validator ran despite a binding failure (%d calls)", validator.calls)
	}
}

func TestBindValidationReceivesDestination(t *testing.T) {
	b := NewASTBinder()
	validator := &stubValidator{}
	b.SetStructValidator(validator)

	var dst validationTarget
	if err := b.Bind(context.Background(), &dst,
		map[string][]string{"name": {"Morgan"}},
		WithValidation(true)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seen, ok := validator.seen.(*validationTarget)
	if !ok {
		t.Fatalf("validator received %T, want *validationTarget", validator.seen)
	}
	if seen != &dst {
		t.Fatal("validator did not receive the caller's destination")
	}
}

func TestBindMapAndBindJSONValidate(t *testing.T) {
	t.Run("BindMap", func(t *testing.T) {
		b := NewASTBinder()
		b.SetStructValidator(&stubValidator{reject: true})

		var dst validationTarget
		err := b.BindMap(context.Background(), &dst,
			map[string]any{"name": "Morgan"},
			IgnoreUnknownKeys(true), WithValidation(true))

		if !errors.Is(err, errInvalid) {
			t.Fatalf("BindMap did not validate, got %v", err)
		}
	})

	t.Run("BindJSON", func(t *testing.T) {
		b := NewASTBinder()
		b.SetStructValidator(&stubValidator{reject: true})

		var dst validationTarget
		err := b.BindJSON(context.Background(), &dst,
			[]byte(`{"name":"Morgan"}`),
			IgnoreUnknownKeys(true), WithValidation(true))

		if !errors.Is(err, errInvalid) {
			t.Fatalf("BindJSON did not validate, got %v", err)
		}
	})
}

func TestBindValidationRunsExactlyOnce(t *testing.T) {
	testCases := []struct {
		name string
		bind func(*ASTBinder, *validationTarget) error
	}{
		{
			name: "Bind",
			bind: func(b *ASTBinder, dst *validationTarget) error {
				return b.Bind(context.Background(), dst,
					map[string][]string{"name": {"Morgan"}}, WithValidation(true))
			},
		},
		{
			name: "BindMap",
			bind: func(b *ASTBinder, dst *validationTarget) error {
				return b.BindMap(context.Background(), dst,
					map[string]any{"name": "Morgan"}, WithValidation(true))
			},
		},
		{
			name: "BindJSON",
			bind: func(b *ASTBinder, dst *validationTarget) error {
				return b.BindJSON(context.Background(), dst,
					[]byte(`{"name":"Morgan"}`), WithValidation(true))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewASTBinder()
			validator := &stubValidator{}
			b.SetStructValidator(validator)

			var dst validationTarget
			if err := tc.bind(b, &dst); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if validator.calls != 1 {
				t.Fatalf("validator ran %d times, want exactly 1", validator.calls)
			}
		})
	}
}

func TestSetStructValidatorNilDisables(t *testing.T) {
	b := NewASTBinder()
	b.SetStructValidator(&stubValidator{reject: true})
	b.SetStructValidator(nil)

	var dst validationTarget
	if err := b.Bind(context.Background(), &dst,
		map[string][]string{"name": {"Morgan"}},
		WithValidation(true)); err != nil {
		t.Fatalf("validation should be disabled, got %v", err)
	}
}

func TestBindValidationErrorIsDescriptive(t *testing.T) {
	b := NewASTBinder()
	b.SetStructValidator(&stubValidator{reject: true})

	var dst validationTarget
	err := b.Bind(context.Background(), &dst,
		map[string][]string{"name": {"Morgan"}},
		WithValidation(true))

	if err == nil || !strings.Contains(err.Error(), "validating bound destination") {
		t.Fatalf("error should identify the validation step, got %v", err)
	}
}
