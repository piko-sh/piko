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

package pml_dto

import (
	"testing"
)

func TestValidationLevel_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		input string
		want  ValidationLevel
	}{
		{input: "strict", want: ValidationStrict},
		{input: "STRICT", want: ValidationStrict},
		{input: "Soft", want: ValidationSoft},
		{input: "skip", want: ValidationSkip},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var v ValidationLevel
			err := v.UnmarshalYAML(func(out any) error {
				*(out.(*string)) = tt.input
				return nil
			})
			if err != nil {
				t.Fatalf("UnmarshalYAML(%q): %v", tt.input, err)
			}
			if v != tt.want {
				t.Errorf("got %q, want %q", v, tt.want)
			}
		})
	}
}

func TestValidationLevel_UnmarshalYAML_Invalid(t *testing.T) {
	var v ValidationLevel
	err := v.UnmarshalYAML(func(out any) error {
		*(out.(*string)) = "invalid"
		return nil
	})
	if err == nil {
		t.Error("expected error for invalid validation level")
	}
}

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.ValidationLevel != ValidationSoft {
		t.Errorf("ValidationLevel = %q, want %q", c.ValidationLevel, ValidationSoft)
	}
	if c.Breakpoint != "480px" {
		t.Errorf("Breakpoint = %q, want %q", c.Breakpoint, "480px")
	}
}
