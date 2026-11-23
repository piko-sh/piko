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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompiledDeploymentNeedsSelfContainedImage(t *testing.T) {
	good := DeploymentReceipt{Files: []string{"app"}, Tags: "piko_embed", SourceDetached: true}
	for _, bad := range []DeploymentReceipt{
		{Files: []string{"app", "dist"}, Tags: good.Tags, SourceDetached: true},
		{Files: good.Files, Tags: "piko_embed_extra", SourceDetached: true},
		{Files: good.Files, Tags: good.Tags},
	} {
		assert.Error(t, checkDeployment(&bad), "invalid deployment accepted: %+v", bad)
	}
	require.NoError(t, checkDeployment(&good))
}
