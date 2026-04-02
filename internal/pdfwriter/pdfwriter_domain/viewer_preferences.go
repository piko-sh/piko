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
	"fmt"
	"strings"
)

// ViewerPreferences controls how PDF viewers display the document.
type ViewerPreferences struct {
	// PageLayout controls how pages are arranged when the document is opened.
	//
	// Valid values: SinglePage, OneColumn, TwoColumnLeft, TwoColumnRight,
	// TwoPageLeft, TwoPageRight. Empty string means viewer default.
	PageLayout string

	// PageMode controls what panel is visible when the document is opened.
	//
	// Valid values: UseNone, UseOutlines, UseThumbs, FullScreen.
	// Empty string means viewer default.
	PageMode string

	// HideToolbar requests that the viewer hide its toolbar.
	HideToolbar bool

	// HideMenubar requests that the viewer hide its menu bar.
	HideMenubar bool

	// HideWindowUI requests that the viewer hide UI elements in the
	// document window (e.g. scroll bars, navigation controls).
	HideWindowUI bool

	// FitWindow requests that the viewer resize its window to fit the
	// first displayed page.
	FitWindow bool

	// CenterWindow requests that the viewer centre its window on screen.
	CenterWindow bool

	// DisplayDocTitle requests that the viewer display the document title
	// from metadata rather than the filename in the title bar.
	DisplayDocTitle bool
}

// buildViewerPreferencesDict writes the /ViewerPreferences dictionary object
// if any boolean preferences are set, and returns the catalog-level entries
// (PageLayout, PageMode, and ViewerPreferences reference).
//
// Takes vp (*ViewerPreferences) which holds the viewer preference settings.
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
//
// Returns an empty string if no viewer preferences are configured.
func buildViewerPreferencesDict(vp *ViewerPreferences, writer *PdfDocumentWriter) string {
	if vp == nil {
		return ""
	}

	var catalogueEntries strings.Builder

	if vp.PageLayout != "" {
		fmt.Fprintf(&catalogueEntries, " /PageLayout /%s", vp.PageLayout)
	}
	if vp.PageMode != "" {
		fmt.Fprintf(&catalogueEntries, " /PageMode /%s", vp.PageMode)
	}

	var prefsEntries []string
	if vp.HideToolbar {
		prefsEntries = append(prefsEntries, "/HideToolbar true")
	}
	if vp.HideMenubar {
		prefsEntries = append(prefsEntries, "/HideMenubar true")
	}
	if vp.HideWindowUI {
		prefsEntries = append(prefsEntries, "/HideWindowUI true")
	}
	if vp.FitWindow {
		prefsEntries = append(prefsEntries, "/FitWindow true")
	}
	if vp.CenterWindow {
		prefsEntries = append(prefsEntries, "/CenterWindow true")
	}
	if vp.DisplayDocTitle {
		prefsEntries = append(prefsEntries, "/DisplayDocTitle true")
	}

	if len(prefsEntries) > 0 {
		prefsNumber := writer.AllocateObject()
		writer.WriteObject(prefsNumber,
			fmt.Sprintf("<< %s >>", strings.Join(prefsEntries, " ")))
		fmt.Fprintf(&catalogueEntries, " /ViewerPreferences %s", FormatReference(prefsNumber))
	}

	return catalogueEntries.String()
}
