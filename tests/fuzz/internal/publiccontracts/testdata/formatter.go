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
package formatter_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/formatter"
)

func TestPublicFormatContract(t *testing.T) {
	s := formatter.NewFormatterService()
	source := []byte("<template><main><p>source sentinel</p></main></template>")
	first, err := s.Format(t.Context(), source)
	require.NoError(t, err)
	second, err := s.Format(t.Context(), first)
	require.NoError(t, err)
	require.True(t, bytes.Contains(first, []byte("source sentinel")) && bytes.Equal(first, second), "PIKO_CHECK formatted-content: missing source text or unstable formatting")
}
