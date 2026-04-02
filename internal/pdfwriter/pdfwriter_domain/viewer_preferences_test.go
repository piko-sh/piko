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
	"strings"
	"testing"
)

func TestBuildViewerPreferencesDict_Nil(t *testing.T) {
	writer := &PdfDocumentWriter{}
	result := buildViewerPreferencesDict(nil, writer)
	if result != "" {
		t.Errorf("expected empty string for nil prefs, got %q", result)
	}
}

func TestBuildViewerPreferencesDict_PageLayoutOnly(t *testing.T) {
	writer := &PdfDocumentWriter{}
	prefs := &ViewerPreferences{PageLayout: "OneColumn"}
	result := buildViewerPreferencesDict(prefs, writer)
	if !strings.Contains(result, "/PageLayout /OneColumn") {
		t.Errorf("expected /PageLayout /OneColumn in result, got %q", result)
	}
	if strings.Contains(result, "/ViewerPreferences") {
		t.Errorf("expected no /ViewerPreferences object when no booleans set, got %q", result)
	}
}

func TestBuildViewerPreferencesDict_PageModeOnly(t *testing.T) {
	writer := &PdfDocumentWriter{}
	prefs := &ViewerPreferences{PageMode: "UseOutlines"}
	result := buildViewerPreferencesDict(prefs, writer)
	if !strings.Contains(result, "/PageMode /UseOutlines") {
		t.Errorf("expected /PageMode /UseOutlines in result, got %q", result)
	}
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
	if !strings.Contains(result, "/ViewerPreferences") {
		t.Fatalf("expected /ViewerPreferences reference in result, got %q", result)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/HideToolbar true") {
		t.Errorf("expected /HideToolbar true in PDF output, got %q", output)
	}
	if !strings.Contains(output, "/FitWindow true") {
		t.Errorf("expected /FitWindow true in PDF output, got %q", output)
	}
	if !strings.Contains(output, "/DisplayDocTitle true") {
		t.Errorf("expected /DisplayDocTitle true in PDF output, got %q", output)
	}
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
	if !strings.Contains(result, "/PageLayout /TwoColumnLeft") {
		t.Errorf("missing /PageLayout, got %q", result)
	}
	if !strings.Contains(result, "/PageMode /FullScreen") {
		t.Errorf("missing /PageMode, got %q", result)
	}
	if !strings.Contains(result, "/ViewerPreferences") {
		t.Errorf("missing /ViewerPreferences, got %q", result)
	}

	output := string(writer.Bytes())
	for _, flag := range []string{"/HideToolbar true", "/HideMenubar true", "/HideWindowUI true", "/FitWindow true", "/CenterWindow true", "/DisplayDocTitle true"} {
		if !strings.Contains(output, flag) {
			t.Errorf("missing %s in PDF output", flag)
		}
	}
}
