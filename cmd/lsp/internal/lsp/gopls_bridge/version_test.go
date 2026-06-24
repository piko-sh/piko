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

package gopls_bridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoplsVersionSupported(t *testing.T) {
	t.Parallel()

	supported := []string{
		"v0.21.0",
		"0.21.0",
		"golang.org/x/tools/gopls v0.21.0",
		"v0.12.0",
		"v1.0.0",
		"v0.30.5-pre",
	}
	for _, version := range supported {
		assert.True(t, goplsVersionSupported(version), "version %q should be supported", version)
	}

	unsupported := []string{
		"v0.11.9",
		"0.5.0",
		"v0.0.1",
	}
	for _, version := range unsupported {
		assert.False(t, goplsVersionSupported(version), "version %q should be rejected", version)
	}

	for _, version := range []string{"", "(devel)", "unknown", "v", "vX.Y.Z", "v-1.20.0", "v0.-3.0"} {
		assert.True(t, goplsVersionSupported(version), "unparseable version %q should be allowed", version)
	}
}

func TestParseGoplsVersion(t *testing.T) {
	t.Parallel()

	major, minor, ok := parseGoplsVersion("golang.org/x/tools/gopls v0.21.3")
	assert.True(t, ok)
	assert.Equal(t, 0, major)
	assert.Equal(t, 21, minor)

	_, _, ok = parseGoplsVersion("no numbers here")
	assert.False(t, ok)
}
