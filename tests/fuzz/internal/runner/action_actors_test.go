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

package runner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestActorFailurePhaseSurvivesInputChanges(t *testing.T) {
	for _, label := range []string{"left", "東京"} {
		input := casefile.ActionValue{Label: label, Quantity: len(label)}
		actors := expectedActors(input)
		actors[1].After = actors[0].After
		err := checkActors(input, actors)
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "action/actor-header:"), "%v", err)
	}
}
