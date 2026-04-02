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

package notification_domain

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotificationBuilder_Do_CancelledContext(t *testing.T) {
	t.Parallel()

	service := newTestService(map[string]*mockProvider{
		"test": {},
	})

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := service.NewNotification().
		Title("Test Alert").
		Provider("test").
		Do(ctx)

	require.ErrorIs(t, err, context.Canceled)
}

func TestNotificationBuilder_Do_ExpiredContext(t *testing.T) {
	t.Parallel()

	service := newTestService(map[string]*mockProvider{
		"test": {},
	})

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	err := service.NewNotification().
		Title("Test Alert").
		Provider("test").
		Do(ctx)

	require.ErrorIs(t, err, context.DeadlineExceeded)
}
