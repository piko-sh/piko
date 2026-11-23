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
package bootstrap

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_adapters"
)

func TestRegisterDirectProbesRegistersTheCSRFProbe(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithCSRFSecret([]byte("0123456789abcdef0123456789abcdef")))
	c.rendererOverride = &stubRenderService{}
	registry := healthprobe_adapters.NewInMemoryRegistry()

	registerDirectProbes(c, registry)

	names := make([]string, 0, len(registry.GetAll()))
	for _, probe := range registry.GetAll() {
		names = append(names, probe.Name())
	}
	assert.Contains(t, names, "CSRFService")
}

func TestRegisterDirectProbesSkipsAFailedCSRFService(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	c.rendererOverride = &stubRenderService{}
	c.csrfOnce.Do(func() { c.csrfErr = errors.New("csrf setup failed") })
	registry := healthprobe_adapters.NewInMemoryRegistry()

	registerDirectProbes(c, registry)

	require.Empty(t, registry.GetAll())
}
