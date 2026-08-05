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
	"fmt"
	"testing"
)

type handWrittenValidator struct{}

func (handWrittenValidator) Struct(s any) error {
	target, ok := s.(*validationTarget)
	if !ok {
		return nil
	}
	if target.Name == "" {
		return errors.New("name must not be empty")
	}
	return nil
}

type customValidatorWithFields struct{ handWrittenValidator }

func (customValidatorWithFields) FieldErrors(err error, destination any) map[string][]string {
	if err == nil {
		return nil
	}
	return map[string][]string{"name": {err.Error()}}
}

func TestCustomValidatorWithoutFieldErrors(t *testing.T) {
	b := NewASTBinder()
	b.SetStructValidator(handWrittenValidator{})

	var dst validationTarget
	err := b.Bind(context.Background(), &dst,
		map[string][]string{"count": {"1"}}, WithValidation(true))

	var failure *ValidationFailedError
	if !errors.As(err, &failure) {
		t.Fatalf("expected a ValidationFailedError, got %T: %v", err, err)
	}
	if failure.Fields != nil {
		t.Fatalf("a validator without FieldErrors should report no field map, got %v", failure.Fields)
	}
	if !errors.Is(err, ErrValidationFailed) || failure.Err == nil {
		t.Fatal("the underlying validator error should be preserved")
	}
	if failure.Err.Error() != "name must not be empty" {
		t.Fatalf("the validator's own error should be preserved unwrapped, got %q", failure.Err.Error())
	}
}

func TestCustomValidatorWithFieldErrors(t *testing.T) {
	b := NewASTBinder()
	b.SetStructValidator(customValidatorWithFields{})

	var dst validationTarget
	err := b.Bind(context.Background(), &dst,
		map[string][]string{"count": {"1"}}, WithValidation(true))

	var failure *ValidationFailedError
	if !errors.As(err, &failure) {
		t.Fatalf("expected a ValidationFailedError, got %T: %v", err, err)
	}
	messages, ok := failure.Fields["name"]
	if !ok || len(messages) == 0 {
		t.Fatalf("expected a field message for %q, got %v", "name", failure.Fields)
	}
	if messages[0] != "name must not be empty" {
		t.Fatalf("unexpected message %q", messages[0])
	}
}

func TestCustomValidatorAcceptsValidInput(t *testing.T) {
	b := NewASTBinder()
	b.SetStructValidator(handWrittenValidator{})

	var dst validationTarget
	if err := b.Bind(context.Background(), &dst,
		map[string][]string{"name": {"Morgan"}}, WithValidation(true)); err != nil {
		t.Fatalf("valid input should bind cleanly, got %v", err)
	}
}

func TestCustomValidatorIsReplaceable(t *testing.T) {
	b := NewASTBinder()
	b.SetStructValidator(handWrittenValidator{})

	var dst validationTarget
	source := map[string][]string{"count": {"1"}}

	if err := b.Bind(context.Background(), &dst, source, WithValidation(true)); err == nil {
		t.Fatal("the first validator should have rejected the empty name")
	}

	b.SetStructValidator(alwaysAccepts{})
	if err := b.Bind(context.Background(), &dst, source, WithValidation(true)); err != nil {
		t.Fatalf("the replacement validator should accept, got %v", err)
	}
}

type alwaysAccepts struct{}

func (alwaysAccepts) Struct(_ any) error { return nil }

func TestCustomValidatorReceivesAnyDestination(t *testing.T) {
	b := NewASTBinder()
	var seen string
	b.SetStructValidator(inspectingValidator{onStruct: func(s any) { seen = fmt.Sprintf("%T", s) }})

	var dst validationTarget
	if err := b.Bind(context.Background(), &dst,
		map[string][]string{"name": {"Morgan"}}, WithValidation(true)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen != "*binder.validationTarget" {
		t.Fatalf("validator saw %q, want the destination pointer", seen)
	}
}

type inspectingValidator struct{ onStruct func(any) }

func (v inspectingValidator) Struct(s any) error {
	v.onStruct(s)
	return nil
}
