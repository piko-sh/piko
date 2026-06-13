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

package worker_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusPartition(t *testing.T) {
	terminal := map[Status]bool{
		StatusCompleted: true, StatusFailed: true, StatusTimeout: true,
		StatusCancelled: true, StatusDiscarded: true,
	}
	for _, status := range []Status{
		StatusPending, StatusScheduled, StatusRunning, StatusCompleted, StatusFailed,
		StatusTimeout, StatusCancelled, StatusRetryable, StatusDiscarded,
	} {
		require.Equalf(t, terminal[status], status.IsTerminal(), "%q IsTerminal", status)
		require.NotEqualf(t, status.IsActive(), status.IsTerminal(),
			"%q must be exactly one of terminal or active", status)
	}
}
