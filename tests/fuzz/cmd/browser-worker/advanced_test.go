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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestAdvancedInteractionClaimsExtendedVocabulary(t *testing.T) {
	for _, operation := range []casefile.ClientOp{
		casefile.ChooseRadio, casefile.SelectMany, casefile.TypeNumber, casefile.RefEdit, casefile.HideMemo,
		casefile.ShowRefocus, casefile.ChildPick, casefile.RemountChild, casefile.SeekTimeline,
	} {
		drive, advanced, err := advancedInteraction("root", casefile.ClientAction{Op: operation, Value: "1"})
		require.NoError(t, err, string(operation))
		assert.True(t, advanced, string(operation))
		assert.NotNil(t, drive, string(operation))
	}
	for _, operation := range []casefile.ClientOp{casefile.SetInput, casefile.ToggleCheck, "unknown"} {
		drive, advanced, err := advancedInteraction("root", casefile.ClientAction{Op: operation})
		require.NoError(t, err, string(operation))
		assert.False(t, advanced, string(operation))
		assert.Nil(t, drive, string(operation))
	}
}
