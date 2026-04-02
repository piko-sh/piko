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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeasureContentExtent_SingleBox(t *testing.T) {
	root := &LayoutBox{
		ContentY:      10,
		ContentHeight: 50,
	}
	assert.InDelta(t, 60.0, MeasureContentExtent(root), 0.001)
}

func TestMeasureContentExtent_NestedBoxes(t *testing.T) {
	root := &LayoutBox{
		ContentY:      0,
		ContentHeight: 100,
		Children: []*LayoutBox{
			{ContentY: 10, ContentHeight: 30},
			{ContentY: 50, ContentHeight: 80},
		},
	}

	assert.InDelta(t, 130.0, MeasureContentExtent(root), 0.001)
}

func TestMeasureContentExtent_DeeplyNested(t *testing.T) {
	root := &LayoutBox{
		ContentY:      0,
		ContentHeight: 20,
		Children: []*LayoutBox{
			{
				ContentY:      5,
				ContentHeight: 10,
				Children: []*LayoutBox{
					{ContentY: 200, ContentHeight: 50},
				},
			},
		},
	}
	assert.InDelta(t, 250.0, MeasureContentExtent(root), 0.001)
}

func TestMeasureContentExtent_WithMargins(t *testing.T) {
	root := &LayoutBox{
		ContentY:      0,
		ContentHeight: 40,
		Margin:        BoxEdges{Bottom: 10},
	}

	assert.InDelta(t, 50.0, MeasureContentExtent(root), 0.001)
}

func TestMeasureContentExtent_EmptyRoot(t *testing.T) {
	root := &LayoutBox{
		ContentY:      0,
		ContentHeight: 0,
	}
	assert.InDelta(t, 0.0, MeasureContentExtent(root), 0.001)
}
