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

package casefile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartialComponentRoutesFollowTheLocale(t *testing.T) {
	english := PartialComponentsInput{Query: "a & b"}
	french := PartialComponentsInput{Query: "a & b", Locale: "fr"}

	assert.Equal(t, "/record-list", english.ListPath())
	assert.Equal(t, "/fr/record-list", french.ListPath())
	assert.Equal(t, "/record-detail?key=amber&q=a+%26+b#summary", english.DetailPath("amber"))
	assert.Equal(t, "/fr/record-detail?key=amber&q=a+%26+b#summary", french.DetailPath("amber"))
}

func TestExpectedPartialComponentsModelsTheJourney(t *testing.T) {
	input := PartialComponentsInput{Labels: [2]string{"one", "two"}, Edited: "edited", Query: "q", Score: 5, EditSecond: true}

	trace := ExpectedPartialComponents(input)

	require.NotEmpty(t, trace.States)
	edit := trace.States[2]
	assert.Equal(t, "edit", edit.Step)
	assert.Equal(t, "edited", edit.Rows[1].Label)
	assert.Equal(t, 5+1+partialEditedScoreBonus, edit.Rows[1].Score)
	removed := trace.States[6]
	assert.Equal(t, "remove", removed.Step)
	require.Len(t, removed.Rows, 1)
	assert.Equal(t, "blue", removed.Rows[0].Key)
	detail := trace.States[len(trace.States)-2]
	assert.Equal(t, "detail", detail.Step)
	assert.Equal(t, -1, detail.ActionRevision)
	assert.Equal(t, "amber / one / q", detail.Detail)
	assert.Equal(t, "back", trace.States[len(trace.States)-1].Step)
}
