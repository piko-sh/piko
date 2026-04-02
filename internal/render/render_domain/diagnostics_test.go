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

package render_domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderDiagnostics_Empty(t *testing.T) {
	var d renderDiagnostics

	assert.Nil(t, d.Warnings)
	assert.Nil(t, d.Errors)
}

func TestRenderDiagnostics_AddWarning(t *testing.T) {
	var d renderDiagnostics

	details := map[string]string{"pageID": "home"}
	d.AddWarning("transformSvg", "SVG too large", details)

	require.Len(t, d.Warnings, 1)
	assert.Equal(t, "transformSvg", d.Warnings[0].Location)
	assert.Equal(t, "SVG too large", d.Warnings[0].Message)
	assert.Equal(t, details, d.Warnings[0].Details)
}

func TestRenderDiagnostics_AddError(t *testing.T) {
	var d renderDiagnostics

	err := errors.New("render failed")
	details := map[string]string{"artefactID": "abc123"}
	d.AddError("buildSpriteSheet", err, "sprite generation failed", details)

	require.Len(t, d.Errors, 1)
	assert.Equal(t, "buildSpriteSheet", d.Errors[0].Location)
	assert.Equal(t, err, d.Errors[0].Err)
	assert.Equal(t, "sprite generation failed", d.Errors[0].Message)
	assert.Equal(t, details, d.Errors[0].Details)
}

func TestRenderDiagnostics_MultipleEntries(t *testing.T) {
	var d renderDiagnostics

	d.AddWarning("loc1", "warn1", nil)
	d.AddWarning("loc2", "warn2", nil)
	d.AddError("loc3", errors.New("err1"), "error1", nil)
	d.AddError("loc4", errors.New("err2"), "error2", nil)

	assert.Len(t, d.Warnings, 2)
	assert.Len(t, d.Errors, 2)

	assert.Equal(t, "loc1", d.Warnings[0].Location)
	assert.Equal(t, "loc2", d.Warnings[1].Location)
	assert.Equal(t, "loc3", d.Errors[0].Location)
	assert.Equal(t, "loc4", d.Errors[1].Location)
}

func TestRenderDiagnostics_AddWarningWithNilDetails(t *testing.T) {
	var d renderDiagnostics

	d.AddWarning("loc", "message", nil)

	require.Len(t, d.Warnings, 1)
	assert.Nil(t, d.Warnings[0].Details)
}

func TestRenderDiagnostics_AddErrorWithNilDetails(t *testing.T) {
	var d renderDiagnostics

	d.AddError("loc", errors.New("fail"), "message", nil)

	require.Len(t, d.Errors, 1)
	assert.Nil(t, d.Errors[0].Details)
}
