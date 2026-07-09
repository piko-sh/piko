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

package pdfwriter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildViewerPreferencesDict_Nil(t *testing.T) {
	writer := &PdfDocumentWriter{}
	result := buildViewerPreferencesDict(nil, writer)
	assert.Equal(t, "", result, "expected empty string for nil prefs")
}

func TestBuildViewerPreferencesDict_PageLayoutOnly(t *testing.T) {
	writer := &PdfDocumentWriter{}
	prefs := &ViewerPreferences{PageLayout: "OneColumn"}
	result := buildViewerPreferencesDict(prefs, writer)
	assert.Contains(t, result, "/PageLayout /OneColumn", "expected /PageLayout /OneColumn in result")
	assert.NotContains(t, result, "/ViewerPreferences", "expected no /ViewerPreferences object when no booleans set")
}

func TestBuildViewerPreferencesDict_PageModeOnly(t *testing.T) {
	writer := &PdfDocumentWriter{}
	prefs := &ViewerPreferences{PageMode: "UseOutlines"}
	result := buildViewerPreferencesDict(prefs, writer)
	assert.Contains(t, result, "/PageMode /UseOutlines", "expected /PageMode /UseOutlines in result")
}

func TestBuildViewerPreferencesDict_BooleanFlags(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	prefs := &ViewerPreferences{
		HideToolbar:     true,
		FitWindow:       true,
		DisplayDocTitle: true,
	}
	result := buildViewerPreferencesDict(prefs, writer)
	require.Contains(t, result, "/ViewerPreferences", "expected /ViewerPreferences reference in result")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/HideToolbar true", "expected /HideToolbar true in PDF output")
	assert.Contains(t, output, "/FitWindow true", "expected /FitWindow true in PDF output")
	assert.Contains(t, output, "/DisplayDocTitle true", "expected /DisplayDocTitle true in PDF output")
}

func TestBuildViewerPreferencesDict_AllFields(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	prefs := &ViewerPreferences{
		PageLayout:      "TwoColumnLeft",
		PageMode:        "FullScreen",
		HideToolbar:     true,
		HideMenubar:     true,
		HideWindowUI:    true,
		FitWindow:       true,
		CenterWindow:    true,
		DisplayDocTitle: true,
	}
	result := buildViewerPreferencesDict(prefs, writer)
	assert.Contains(t, result, "/PageLayout /TwoColumnLeft", "missing /PageLayout")
	assert.Contains(t, result, "/PageMode /FullScreen", "missing /PageMode")
	assert.Contains(t, result, "/ViewerPreferences", "missing /ViewerPreferences")

	output := string(writer.Bytes())
	for _, flag := range []string{"/HideToolbar true", "/HideMenubar true", "/HideWindowUI true", "/FitWindow true", "/CenterWindow true", "/DisplayDocTitle true"} {
		assert.Contains(t, output, flag, "missing %s in PDF output", flag)
	}
}
