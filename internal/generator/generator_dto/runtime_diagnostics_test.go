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

package generator_dto

import "testing"

func TestSeverity_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		sev  Severity
	}{
		{name: "debug", sev: Debug, want: "debug"},
		{name: "info", sev: Info, want: "info"},
		{name: "warning", sev: Warning, want: "warning"},
		{name: "error", sev: Error, want: "error"},
		{name: "unknown", sev: Severity(99), want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.sev.String(); got != tt.want {
				t.Errorf("Severity(%d).String() = %q, want %q", tt.sev, got, tt.want)
			}
		})
	}
}

func TestSeverity_CodeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		sev  Severity
	}{
		{name: "Debug", sev: Debug, want: "Debug"},
		{name: "Info", sev: Info, want: "Info"},
		{name: "Warning", sev: Warning, want: "Warning"},
		{name: "Error", sev: Error, want: "Error"},
		{name: "unknown", sev: Severity(99), want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.sev.CodeString(); got != tt.want {
				t.Errorf("Severity(%d).CodeString() = %q, want %q", tt.sev, got, tt.want)
			}
		})
	}
}
