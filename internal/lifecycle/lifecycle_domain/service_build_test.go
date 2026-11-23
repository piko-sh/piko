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

package lifecycle_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInterpretedManifestStoreViewAdapterDelegatesErrorPagesAndPreviews(t *testing.T) {
	t.Parallel()

	adapter := &interpretedManifestStoreViewAdapter{runner: &mockInterpretedRunner{}}

	entry, found := adapter.FindErrorPage(404, "/missing")
	assert.False(t, found)
	assert.Nil(t, entry)
	assert.Empty(t, adapter.ListPreviewEntries())
}

func TestSetCurrentRunnerSeedsTheRunnerSnapshot(t *testing.T) {
	t.Parallel()

	service := &lifecycleService{}
	assert.Nil(t, service.currentRunnerSnapshot())

	runner := &mockInterpretedRunner{keys: []string{"page"}}
	service.SetCurrentRunner(runner)

	assert.Same(t, runner, service.currentRunnerSnapshot())
}
