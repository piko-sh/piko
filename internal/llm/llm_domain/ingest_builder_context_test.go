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

package llm_domain

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIngestBuilder_Do_CancelledContext(t *testing.T) {
	t.Parallel()

	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{{ID: "doc1", Content: "hello"}},
	})

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := builder.Do(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestIngestBuilder_Do_ExpiredContext(t *testing.T) {
	t.Parallel()

	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	builder := NewIngestBuilder(service, "ns")
	builder.Loader(&mockLoaderPort{
		docs: []Document{{ID: "doc1", Content: "hello"}},
	})

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	err := builder.Do(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
