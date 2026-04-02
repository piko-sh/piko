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

package layouter_adapters

// Implements a mock ImageResolverPort for testing that returns fixed
// dimensions for any image source. Uses the function-field pattern.

import "context"

const (
	// mockImageWidth is the default width in points returned by
	// MockImageResolver when no override function is set.
	mockImageWidth = 100.0

	// mockImageHeight is the default height in points returned by
	// MockImageResolver when no override function is set.
	mockImageHeight = 100.0
)

// MockImageResolver is a test double for ImageResolverPort that returns fixed
// image dimensions. The function field can be overridden; nil uses the
// default (100x100 points).
type MockImageResolver struct {
	// GetImageDimensionsFunc is an optional override for the
	// GetImageDimensions method. When nil, the default 100x100
	// point dimensions are returned.
	GetImageDimensionsFunc func(ctx context.Context, source string) (width, height float64, err error)
}

// GetImageDimensions returns the intrinsic dimensions for the given image
// source. Default: 100x100 points.
//
// Takes source (string) which is the image source path or URL.
//
// Returns width and height in points, or an error.
func (m *MockImageResolver) GetImageDimensions(ctx context.Context, source string) (width float64, height float64, err error) {
	if m.GetImageDimensionsFunc != nil {
		return m.GetImageDimensionsFunc(ctx, source)
	}
	return mockImageWidth, mockImageHeight, nil
}
