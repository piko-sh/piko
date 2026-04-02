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

package component_domain

import (
	"testing"
)

func TestValidateTagName(t *testing.T) {
	testCases := []struct {
		name         string
		tagName      string
		errorMessage string
		wantError    bool
	}{

		{
			name:      "valid simple component",
			tagName:   "my-button",
			wantError: false,
		},
		{
			name:      "valid component with multiple hyphens",
			tagName:   "my-awesome-button",
			wantError: false,
		},
		{
			name:      "valid component with numbers",
			tagName:   "button-v2",
			wantError: false,
		},
		{
			name:      "valid component with uppercase",
			tagName:   "My-Button",
			wantError: false,
		},
		{
			name:      "valid uikit style component",
			tagName:   "uikit-card",
			wantError: false,
		},

		{
			name:         "invalid no hyphen",
			tagName:      "button",
			wantError:    true,
			errorMessage: "must contain a hyphen",
		},
		{
			name:         "invalid single word",
			tagName:      "modal",
			wantError:    true,
			errorMessage: "must contain a hyphen",
		},

		{
			name:         "invalid empty string",
			tagName:      "",
			wantError:    true,
			errorMessage: "cannot be empty",
		},

		{
			name:         "invalid shadows div",
			tagName:      "div",
			wantError:    true,
			errorMessage: "must contain a hyphen",
		},
		{
			name:         "invalid shadows span",
			tagName:      "span",
			wantError:    true,
			errorMessage: "must contain a hyphen",
		},
		{
			name:         "invalid shadows button",
			tagName:      "button",
			wantError:    true,
			errorMessage: "must contain a hyphen",
		},
		{
			name:         "invalid shadows slot",
			tagName:      "slot",
			wantError:    true,
			errorMessage: "must contain a hyphen",
		},
		{
			name:         "invalid shadows template",
			tagName:      "template",
			wantError:    true,
			errorMessage: "must contain a hyphen",
		},

		{
			name:         "invalid piko prefix",
			tagName:      "piko:slot",
			wantError:    true,
			errorMessage: "reserved prefix 'piko:'",
		},
		{
			name:         "invalid piko svg",
			tagName:      "piko:svg",
			wantError:    true,
			errorMessage: "reserved prefix 'piko:'",
		},
		{
			name:         "invalid pml prefix",
			tagName:      "pml-button",
			wantError:    true,
			errorMessage: "reserved prefix 'pml-'",
		},
		{
			name:         "invalid PML uppercase",
			tagName:      "PML-Card",
			wantError:    true,
			errorMessage: "reserved prefix 'pml-'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTagName(tc.tagName)

			if tc.wantError {
				if err == nil {
					t.Errorf("ValidateTagName(%q) = nil, want error containing %q", tc.tagName, tc.errorMessage)
					return
				}
				if tc.errorMessage != "" && !containsString(err.Error(), tc.errorMessage) {
					t.Errorf("ValidateTagName(%q) = %q, want error containing %q", tc.tagName, err.Error(), tc.errorMessage)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTagName(%q) = %v, want nil", tc.tagName, err)
				}
			}
		})
	}
}

func TestIsHTMLElement(t *testing.T) {
	testCases := []struct {
		name    string
		tagName string
		want    bool
	}{
		{name: "div", tagName: "div", want: true},
		{name: "span", tagName: "span", want: true},
		{name: "button", tagName: "button", want: true},
		{name: "input", tagName: "input", want: true},
		{name: "form", tagName: "form", want: true},
		{name: "table", tagName: "table", want: true},
		{name: "slot", tagName: "slot", want: true},
		{name: "template", tagName: "template", want: true},
		{name: "uppercase DIV", tagName: "DIV", want: true},
		{name: "mixed case Div", tagName: "Div", want: true},
		{name: "custom element", tagName: "my-button", want: false},
		{name: "unknown element", tagName: "foobar", want: false},
		{name: "piko tag", tagName: "piko:slot", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsHTMLElement(tc.tagName)
			if got != tc.want {
				t.Errorf("IsHTMLElement(%q) = %v, want %v", tc.tagName, got, tc.want)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
