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

package inspector_dto

import "testing"

func TestFunctionSignature_ToSignatureString(t *testing.T) {
	tests := []struct {
		name string
		want string
		sig  FunctionSignature
	}{
		{
			name: "no params or returns",
			want: "func() ",
			sig:  FunctionSignature{},
		},
		{
			name: "single return",
			want: "func(int, string) error",
			sig:  FunctionSignature{Params: []string{"int", "string"}, Results: []string{"error"}},
		},
		{
			name: "multiple returns",
			want: "func(string) (int, error)",
			sig:  FunctionSignature{Params: []string{"string"}, Results: []string{"int", "error"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sig.ToSignatureString(); got != tt.want {
				t.Errorf("ToSignatureString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseStructTag(t *testing.T) {
	raw := "`prop:\"title\" validate:\"required\" json:\"title\"`"
	got := ParseStructTag(raw)

	if got["prop"] != "title" {
		t.Errorf("prop = %q, want %q", got["prop"], "title")
	}
	if got["validate"] != "required" {
		t.Errorf("validate = %q, want %q", got["validate"], "required")
	}

	if _, exists := got["json"]; exists {
		t.Error("json should not be in parsed Piko tags")
	}
}

func TestParseStructTag_Empty(t *testing.T) {
	got := ParseStructTag("")
	if len(got) != 0 {
		t.Errorf("empty tag should return empty map, got %v", got)
	}
}
