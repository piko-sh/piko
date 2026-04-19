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

package templater_domain

import "sync/atomic"

// MockManifestStoreView is a test double for ManifestStoreView that
// returns zero values from nil function fields and tracks call counts
// atomically.
type MockManifestStoreView struct {
	// GetKeysFunc is the function called by GetKeys.
	GetKeysFunc func() []string

	// GetPageEntryFunc is the function called by GetPageEntry.
	GetPageEntryFunc func(path string) (PageEntryView, bool)

	// FindErrorPageFunc is the function called by FindErrorPage.
	FindErrorPageFunc func(statusCode int, requestPath string) (PageEntryView, bool)

	// ListPreviewEntriesFunc is the function called by ListPreviewEntries.
	ListPreviewEntriesFunc func() []PreviewCatalogueEntry

	// GetKeysCallCount tracks how many times GetKeys was called.
	GetKeysCallCount int64

	// GetPageEntryCallCount tracks how many times GetPageEntry was called.
	GetPageEntryCallCount int64

	// FindErrorPageCallCount tracks how many times FindErrorPage was called.
	FindErrorPageCallCount int64

	// ListPreviewEntriesCallCount tracks how many times ListPreviewEntries
	// was called.
	ListPreviewEntriesCallCount int64
}

var _ ManifestStoreView = (*MockManifestStoreView)(nil)

// GetKeys returns all component source paths in the manifest.
//
// Returns []string, or nil if GetKeysFunc is nil.
func (m *MockManifestStoreView) GetKeys() []string {
	atomic.AddInt64(&m.GetKeysCallCount, 1)
	if m.GetKeysFunc != nil {
		return m.GetKeysFunc()
	}
	return nil
}

// GetPageEntry retrieves the view for a component by its source path.
//
// Takes path (string) which is the component source path to look up.
//
// Returns (PageEntryView, bool), or (nil, false) if GetPageEntryFunc is nil.
func (m *MockManifestStoreView) GetPageEntry(path string) (PageEntryView, bool) {
	atomic.AddInt64(&m.GetPageEntryCallCount, 1)
	if m.GetPageEntryFunc != nil {
		return m.GetPageEntryFunc(path)
	}
	return nil, false
}

// FindErrorPage looks up the most specific error page for the given status code
// and request path.
//
// Takes statusCode (int) which is the HTTP status code to find an error page for.
// Takes requestPath (string) which is the request path to match against.
//
// Returns (PageEntryView, bool), or (nil, false) if FindErrorPageFunc is nil.
func (m *MockManifestStoreView) FindErrorPage(statusCode int, requestPath string) (PageEntryView, bool) {
	atomic.AddInt64(&m.FindErrorPageCallCount, 1)
	if m.FindErrorPageFunc != nil {
		return m.FindErrorPageFunc(statusCode, requestPath)
	}
	return nil, false
}

// ListPreviewEntries returns all entries that have a Preview function.
//
// Returns []PreviewCatalogueEntry, or nil if ListPreviewEntriesFunc is nil.
func (m *MockManifestStoreView) ListPreviewEntries() []PreviewCatalogueEntry {
	atomic.AddInt64(&m.ListPreviewEntriesCallCount, 1)
	if m.ListPreviewEntriesFunc != nil {
		return m.ListPreviewEntriesFunc()
	}
	return nil
}
