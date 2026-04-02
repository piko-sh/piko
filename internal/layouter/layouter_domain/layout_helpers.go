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

package layouter_domain

import "math"

// clampStretchDimension applies min/max constraints to a
// stretched content size. When isWidth is true the width
// constraints are used; otherwise height constraints apply.
//
// This is shared by both flex and grid layout when
// stretching items along the cross axis.
//
// Takes size (float64) which is the content-box size
// before clamping.
// Takes style (*ComputedStyle) which is the item style
// carrying min/max constraints.
// Takes isWidth (bool) which selects width or height
// constraints.
//
// Returns the clamped content-box size.
func clampStretchDimension(size float64, style *ComputedStyle, isWidth bool) float64 {
	if isWidth {
		if !style.MinWidth.IsAuto() && !style.MinWidth.IsFitContent() {
			size = math.Max(size, adjustForBoxSizing(style.MinWidth.Resolve(0, 0), style, true))
		}
		if !style.MaxWidth.IsAuto() && !style.MaxWidth.IsFitContent() {
			size = math.Min(size, adjustForBoxSizing(style.MaxWidth.Resolve(0, 0), style, true))
		}
	} else {
		if !style.MinHeight.IsAuto() && !style.MinHeight.IsFitContent() {
			size = math.Max(size, adjustForBoxSizing(style.MinHeight.Resolve(0, 0), style, false))
		}
		if !style.MaxHeight.IsAuto() && !style.MaxHeight.IsFitContent() {
			size = math.Min(size, adjustForBoxSizing(style.MaxHeight.Resolve(0, 0), style, false))
		}
	}
	return size
}
