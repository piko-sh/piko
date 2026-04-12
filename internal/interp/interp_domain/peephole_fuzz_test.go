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

//go:build fuzz

package interp_domain

import (
	"context"
	"testing"
)

func FuzzGvnDoesNotPanic(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4})
	f.Add([]byte{})
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, raw []byte) {
		body := decodeFuzzBody(raw)
		cf := &CompiledFunction{body: body}
		_ = cf.runFunctionGvn(context.Background())
		if len(cf.body) != len(body) {
			t.Fatalf("GVN changed body length: got %d, want %d", len(cf.body), len(body))
		}
	})
}

func FuzzBceDoesNotPanic(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4})
	f.Add([]byte{})
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, raw []byte) {
		body := decodeFuzzBody(raw)
		cf := &CompiledFunction{body: body}
		cf.elideRedundantBoundsChecks(cf.body)
		if len(cf.body) != len(body) {
			t.Fatalf("BCE changed body length: got %d, want %d", len(cf.body), len(body))
		}
	})
}

func FuzzPointerAliasDoesNotPanic(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4})
	f.Add([]byte{})
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, raw []byte) {
		body := decodeFuzzBody(raw)
		cf := &CompiledFunction{body: body}
		original := append([]instruction(nil), cf.body...)
		_ = runPointerAliasAnalysis(context.Background(), cf)
		if len(cf.body) != len(original) {
			t.Fatalf("alias analysis mutated body length: got %d, want %d", len(cf.body), len(original))
		}
		for i := range cf.body {
			if cf.body[i] != original[i] {
				t.Fatalf("alias analysis mutated body[%d]: got %#v, want %#v", i, cf.body[i], original[i])
			}
		}
	})
}

func decodeFuzzBody(raw []byte) []instruction {
	const wordSize = 4
	const maxWords = 1024
	count := min(len(raw)/wordSize, maxWords)
	body := make([]instruction, count)
	for i := range count {
		base := i * wordSize
		body[i] = instruction{
			op: opcode(raw[base]),
			a:  raw[base+1],
			b:  raw[base+2],
			c:  raw[base+3],
		}
	}
	return body
}
