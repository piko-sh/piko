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

package cssinliner

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	parseCacheTestCSS = `.alpha { color: red; padding: 4px } .beta { color: blue; margin: 8px }`
)

func TestParseCache_CacheHitIdentical(t *testing.T) {
	t.Parallel()

	p := newTestProcessor(newPassthroughResolver())
	fs := newTestFSReader()
	loc := ast_domain.Location{Line: 1, Column: 1}

	out1, _, err1 := p.Process(context.Background(), parseCacheTestCSS, "one.css", loc, fs)
	require.NoError(t, err1)
	require.NotEmpty(t, out1)

	out2, _, err2 := p.Process(context.Background(), parseCacheTestCSS, "two.css", loc, fs)
	require.NoError(t, err2)

	assert.Equal(t, out1, out2, "a cached parse must yield identical output to the first parse")
}

func TestParseCache_ConcurrentIdentical(t *testing.T) {
	t.Parallel()

	p := newTestProcessor(newPassthroughResolver())
	fs := newTestFSReader()
	loc := ast_domain.Location{Line: 1, Column: 1}

	const n = 32
	results := make([]string, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			out, _, err := p.Process(context.Background(), parseCacheTestCSS, "p.css", loc, fs)
			results[i], errs[i] = out, err
		})
	}
	wg.Wait()

	for i := range n {
		require.NoError(t, errs[i])
		assert.Equal(t, results[0], results[i], "all concurrent results must be identical")
	}
}
