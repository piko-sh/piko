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

package session_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSession_ImportPersistsAcrossSubmits(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, `import "strings"`)
	require.Equal(t, "HI", mustSubmit(t, sess, `strings.ToUpper("hi")`))
}

func TestSession_DuplicateImport_NoOp(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, `import "strings"`)
	mustSubmit(t, sess, `import "strings"`)
	require.Equal(t, "HI", mustSubmit(t, sess, `strings.ToUpper("hi")`))
}

func TestSession_MultipleImportsInOneSubmit(t *testing.T) {
	t.Parallel()
	sess := newTestSession(t)
	mustSubmit(t, sess, `import (
	"strings"
	"fmt"
)`)
	require.Equal(t, "HI", mustSubmit(t, sess, `strings.ToUpper("hi")`))
	require.Equal(t, "x=7", mustSubmit(t, sess, `fmt.Sprintf("x=%d", 7)`))
}
