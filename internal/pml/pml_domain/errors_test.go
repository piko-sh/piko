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

package pml_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func Test_newError(t *testing.T) {
	location := ast_domain.Location{Line: 10, Column: 5}
	err := newError("Test message", "pml-button", SeverityError, location)

	require.NotNil(t, err)
	assert.Equal(t, "Test message", err.Message)
	assert.Equal(t, "pml-button", err.TagName)
	assert.Equal(t, SeverityError, err.Severity)
	assert.Equal(t, 10, err.Location.Line)
	assert.Equal(t, 5, err.Location.Column)
}

func TestError_Error_ErrorSeverity(t *testing.T) {
	err := &Error{
		Message:  "Invalid attribute value",
		TagName:  "pml-button",
		Severity: SeverityError,
		Location: ast_domain.Location{Line: 15, Column: 20},
	}

	errorString := err.Error()

	assert.Contains(t, errorString, "PikoML")
	assert.Contains(t, errorString, "error")
	assert.Contains(t, errorString, "<pml-button>")
	assert.Contains(t, errorString, "L15:C20")
	assert.Contains(t, errorString, "Invalid attribute value")
}

func TestError_Error_WarningSeverity(t *testing.T) {
	err := &Error{
		Message:  "Deprecated attribute",
		TagName:  "pml-img",
		Severity: SeverityWarning,
		Location: ast_domain.Location{Line: 5, Column: 1},
	}

	errorString := err.Error()

	assert.Contains(t, errorString, "warning")
	assert.Contains(t, errorString, "<pml-img>")
	assert.Contains(t, errorString, "L5:C1")
	assert.Contains(t, errorString, "Deprecated attribute")
}

func TestSeverity_Values(t *testing.T) {
	testCases := []struct {
		severity Severity
		expected string
	}{
		{severity: SeverityError, expected: "error"},
		{severity: SeverityWarning, expected: "warning"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.severity))
		})
	}
}

func TestError_Error_ZeroLocation(t *testing.T) {
	err := &Error{
		Message:  "Some error",
		TagName:  "pml-row",
		Severity: SeverityError,
		Location: ast_domain.Location{},
	}

	errorString := err.Error()

	assert.Contains(t, errorString, "L0:C0")
	assert.Contains(t, errorString, "Some error")
}
