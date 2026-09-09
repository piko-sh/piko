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

package inspector_domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/wasm/wasm_data"
)

func TestStdlibPackageListsAgree(t *testing.T) {
	t.Parallel()

	assert.Equal(t, inspector_domain.StdlibPackages, wasm_data.DefaultStdlibPackages,
		"internal/wasm/wasm_data.DefaultStdlibPackages must mirror "+
			"inspector_domain.StdlibPackages, otherwise the embedded stdlib bundle and the "+
			"live type index disagree about which packages exist")
}
