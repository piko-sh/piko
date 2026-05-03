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

//go:build js && wasm

package main

import (
	"syscall/js"
	"testing"
)

func hasThenMethod(value js.Value) bool {
	if value.IsUndefined() || value.IsNull() {
		return false
	}
	if value.Type() != js.TypeObject {
		return false
	}
	then := value.Get("then")
	return then.Type() == js.TypeFunction
}

func TestRejectedPromise_ReturnsThenable(t *testing.T) {
	promise := rejectedPromise("test rejection message")
	if !hasThenMethod(promise) {
		t.Fatalf("rejectedPromise must return a thenable; got Type=%v", promise.Type())
	}
}

func TestParseRequestSafely_MissingArgumentReturnsErrorMessage(t *testing.T) {
	var target struct {
		Source string `json:"source"`
	}

	errMessage := parseRequestSafely("piko.test", nil, &target, "test requires a request object")
	if errMessage == "" {
		t.Fatal("parseRequestSafely with no arguments must return a non-empty error message")
	}
	if errMessage != "test requires a request object" {
		t.Errorf("errMessage = %q, want %q", errMessage, "test requires a request object")
	}
}

func TestParseRequestSafely_ValidRequestReturnsEmpty(t *testing.T) {
	var target struct {
		Source string `json:"source"`
	}

	requestObject := js.Global().Get("JSON").Call("parse", `{"source":"hi"}`)
	errMessage := parseRequestSafely("piko.test", []js.Value{requestObject}, &target, "test requires a request object")
	if errMessage != "" {
		t.Fatalf("parseRequestSafely with valid request must return empty; got %q", errMessage)
	}
	if target.Source != "hi" {
		t.Errorf("target.Source = %q, want %q", target.Source, "hi")
	}
}
