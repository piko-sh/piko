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

package notification_domain

import (
	"errors"
	"strings"
	"testing"
)

func TestMultiError_Error_Empty(t *testing.T) {
	me := &MultiError{}
	if me.Error() != "no errors" {
		t.Errorf("expected %q, got %q", "no errors", me.Error())
	}
}

func TestMultiError_Error_Single(t *testing.T) {
	me := &MultiError{Errors: []error{errors.New("single failure")}}
	if me.Error() != "single failure" {
		t.Errorf("expected %q, got %q", "single failure", me.Error())
	}
}

func TestMultiError_Error_Multiple(t *testing.T) {
	me := &MultiError{Errors: []error{
		errors.New("first"),
		errors.New("second"),
		errors.New("third"),
	}}
	got := me.Error()
	if !strings.HasPrefix(got, "3 errors occurred: ") {
		t.Errorf("expected prefix %q, got %q", "3 errors occurred: ", got)
	}
	if !strings.Contains(got, "first") || !strings.Contains(got, "second") || !strings.Contains(got, "third") {
		t.Errorf("expected all error messages in output, got %q", got)
	}
	if !strings.Contains(got, "; ") {
		t.Errorf("expected semicolon separator, got %q", got)
	}
}

func TestMultiError_Unwrap(t *testing.T) {
	errs := []error{errors.New("a"), errors.New("b")}
	me := &MultiError{Errors: errs}
	unwrapped := me.Unwrap()
	if len(unwrapped) != 2 {
		t.Fatalf("expected 2 unwrapped errors, got %d", len(unwrapped))
	}
	if !errors.Is(unwrapped[0], errs[0]) || !errors.Is(unwrapped[1], errs[1]) {
		t.Error("unwrapped errors do not match original")
	}
}

func TestMultiError_HasErrors_True(t *testing.T) {
	me := &MultiError{Errors: []error{errors.New("err")}}
	if !me.HasErrors() {
		t.Error("expected HasErrors to return true")
	}
}

func TestMultiError_HasErrors_False(t *testing.T) {
	me := &MultiError{}
	if me.HasErrors() {
		t.Error("expected HasErrors to return false")
	}
}

func TestProviderError_Error(t *testing.T) {
	pe := &ProviderError{Provider: "slack", Err: errors.New("timeout")}
	expected := `provider "slack": timeout`
	if pe.Error() != expected {
		t.Errorf("expected %q, got %q", expected, pe.Error())
	}
}

func TestProviderError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	pe := &ProviderError{Provider: "test", Err: inner}
	if !errors.Is(inner, pe.Unwrap()) {
		t.Error("Unwrap did not return the inner error")
	}
}

func TestProviderError_ErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	pe := &ProviderError{Provider: "test", Err: sentinel}
	if !errors.Is(pe, sentinel) {
		t.Error("errors.Is should find the sentinel through ProviderError")
	}
}

func TestMultiError_ErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	me := &MultiError{Errors: []error{errors.New("other"), sentinel}}
	if !errors.Is(me, sentinel) {
		t.Error("errors.Is should find the sentinel through MultiError.Unwrap")
	}
}
