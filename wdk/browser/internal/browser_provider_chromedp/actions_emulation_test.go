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

package browser_provider_chromedp

import (
	"testing"
)

func TestCommonViewportSizes_AllEntriesExist(t *testing.T) {
	expectedNames := []string{
		"mobile-s", "mobile-m", "mobile-l", "tablet",
		"laptop", "laptop-l", "desktop", "desktop-4k",
	}

	for _, name := range expectedNames {
		t.Run(name, func(t *testing.T) {
			if _, ok := CommonViewportSizes[name]; !ok {
				t.Errorf("CommonViewportSizes missing %q", name)
			}
		})
	}

	if len(CommonViewportSizes) != len(expectedNames) {
		t.Errorf("CommonViewportSizes has %d entries, expected %d", len(CommonViewportSizes), len(expectedNames))
	}
}

func TestCommonViewportSizes_SpecificDimensions(t *testing.T) {
	testCases := []struct {
		name           string
		expectedWidth  int64
		expectedHeight int64
	}{
		{name: "mobile-s", expectedWidth: 320, expectedHeight: 568},
		{name: "mobile-m", expectedWidth: 375, expectedHeight: 667},
		{name: "mobile-l", expectedWidth: 414, expectedHeight: 896},
		{name: "tablet", expectedWidth: 768, expectedHeight: 1024},
		{name: "laptop", expectedWidth: 1024, expectedHeight: 768},
		{name: "laptop-l", expectedWidth: 1440, expectedHeight: 900},
		{name: "desktop", expectedWidth: 1920, expectedHeight: 1080},
		{name: "desktop-4k", expectedWidth: 3840, expectedHeight: 2160},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := CommonViewportSizes[tc.name]
			if size.Width != tc.expectedWidth {
				t.Errorf("Width = %d, expected %d", size.Width, tc.expectedWidth)
			}
			if size.Height != tc.expectedHeight {
				t.Errorf("Height = %d, expected %d", size.Height, tc.expectedHeight)
			}
		})
	}
}

func TestViewportConstants(t *testing.T) {
	if ViewportWidthHD != 1920 {
		t.Errorf("ViewportWidthHD = %d, expected 1920", ViewportWidthHD)
	}
	if ViewportHeightHD != 1080 {
		t.Errorf("ViewportHeightHD = %d, expected 1080", ViewportHeightHD)
	}
	if ViewportWidth4K != 3840 {
		t.Errorf("ViewportWidth4K = %d, expected 3840", ViewportWidth4K)
	}
	if ViewportHeight4K != 2160 {
		t.Errorf("ViewportHeight4K = %d, expected 2160", ViewportHeight4K)
	}
}

func TestEmulateViewportByName_UnknownSize(t *testing.T) {
	ctx := &ActionContext{}
	err := EmulateViewportByName(ctx, "nonexistent-size")
	if err == nil {
		t.Fatal("expected error for unknown viewport size")
	}
}
