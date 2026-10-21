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

package coordinator_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
)

func TestPendingBuildsAreServedFirstInFirstOut(t *testing.T) {
	t.Parallel()

	service := newTargetedTestService(&mockAnnotator{})
	request := func(path, causation string) *coordinator_dto.BuildRequest {
		return &coordinator_dto.BuildRequest{
			CausationID: causation,
			EntryPoints: []annotator_dto.EntryPoint{{Path: path}},
		}
	}

	for _, queued := range []*coordinator_dto.BuildRequest{
		request("pages/a.pk", "a-first"),
		request("pages/b.pk", "b"),
		request("pages/c.pk", "c"),
		request("pages/a.pk", "a-second"),
		request("pages/d.pk", "d"),
	} {
		service.enqueuePendingBuild(context.Background(), queued)
	}

	var served []string
	for {
		next, found := service.takeNextPendingBuild()
		if !found {
			break
		}
		served = append(served, next.CausationID)
	}

	assert.Equal(t, []string{"a-second", "b", "c", "d"}, served,
		"requests are served in arrival order; a newer request for a queued target replaces it in place")

	service.enqueuePendingBuild(context.Background(), request("pages/e.pk", "e"))
	service.drainRebuildTrigger(context.Background())
	_, found := service.takeNextPendingBuild()
	require.False(t, found, "invalidation discards the queue and its order")
}
