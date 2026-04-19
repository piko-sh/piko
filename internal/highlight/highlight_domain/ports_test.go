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

package highlight_domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/highlight/highlight_domain"
)

type stubHighlighter struct {
	lastCode     string
	lastLanguage string
	output       string
}

func (s *stubHighlighter) Highlight(code, language string) string {
	s.lastCode = code
	s.lastLanguage = language
	return s.output
}

func TestHighlighter_Interface_RecordsArgumentsAndReturnsOutput(t *testing.T) {
	t.Parallel()

	stub := &stubHighlighter{output: "<pre><code>echo</code></pre>"}

	var port highlight_domain.Highlighter = stub

	got := port.Highlight("echo", "go")

	require.Equal(t, "<pre><code>echo</code></pre>", got)
	require.Equal(t, "echo", stub.lastCode)
	require.Equal(t, "go", stub.lastLanguage)
}

func TestHighlighter_Interface_HandlesEmptyInputs(t *testing.T) {
	t.Parallel()

	stub := &stubHighlighter{output: ""}

	got := stub.Highlight("", "")

	require.Equal(t, "", got)
	require.Equal(t, "", stub.lastCode)
	require.Equal(t, "", stub.lastLanguage)
}
