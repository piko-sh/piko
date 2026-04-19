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

package registry_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func profileWith(capability string, keyValues ...string) *DesiredProfile {
	profile := &DesiredProfile{}
	profile.CapabilityName = capability
	for i := 0; i+1 < len(keyValues); i += 2 {
		profile.Params.SetByName(keyValues[i], keyValues[i+1])
	}
	return profile
}

func TestComputeVariantFingerprint_StableAndParamOrderIndependent(t *testing.T) {
	first := ComputeVariantFingerprint("src1", profileWith("image", "width", "240", "format", "webp"))
	second := ComputeVariantFingerprint("src1", profileWith("image", "format", "webp", "width", "240"))
	require.NotEmpty(t, first, "fingerprint must be non-empty")
	assert.Equal(t, first, second, "fingerprint must be independent of parameter ordering")
}

func TestComputeVariantFingerprint_ChangesWithInputs(t *testing.T) {
	base := ComputeVariantFingerprint("src1", profileWith("image", "width", "240"))
	changed := map[string]string{
		"different source content": ComputeVariantFingerprint("src2", profileWith("image", "width", "240")),
		"different capability":     ComputeVariantFingerprint("src1", profileWith("video", "width", "240")),
		"different param value":    ComputeVariantFingerprint("src1", profileWith("image", "width", "480")),
		"additional param":         ComputeVariantFingerprint("src1", profileWith("image", "width", "240", "quality", "80")),
	}
	for name, fingerprint := range changed {
		assert.NotEqual(t, base, fingerprint, "a %s must change the fingerprint", name)
	}
}
