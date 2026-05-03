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

package orchestrator_dal_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	t.Parallel()

	all := []error{
		orchestrator_dal.ErrTaskNotFound,
		orchestrator_dal.ErrWorkflowNotFound,
		orchestrator_dal.ErrTransactionFailed,
		orchestrator_dal.ErrDatabaseClosed,
	}

	for outerIndex, outer := range all {
		for innerIndex, inner := range all {
			if outerIndex == innerIndex {
				continue
			}
			require.NotEqualf(t, outer, inner, "errors at index %d and %d should differ", outerIndex, innerIndex)
		}
	}
}

func TestSentinelErrors_HaveDescriptiveMessages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want string
	}{
		{orchestrator_dal.ErrTaskNotFound, "task not found"},
		{orchestrator_dal.ErrWorkflowNotFound, "workflow not found"},
		{orchestrator_dal.ErrTransactionFailed, "transaction failed"},
		{orchestrator_dal.ErrDatabaseClosed, "database connection is closed"},
	}

	for _, tc := range cases {
		require.Equal(t, tc.want, tc.err.Error())
	}
}

func TestSentinelErrors_WorkWithErrorsIs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		base    error
		wrapper error
	}{
		{orchestrator_dal.ErrTaskNotFound, fmt.Errorf("looking up task: %w", orchestrator_dal.ErrTaskNotFound)},
		{orchestrator_dal.ErrWorkflowNotFound, fmt.Errorf("looking up workflow: %w", orchestrator_dal.ErrWorkflowNotFound)},
		{orchestrator_dal.ErrTransactionFailed, fmt.Errorf("commit: %w", orchestrator_dal.ErrTransactionFailed)},
		{orchestrator_dal.ErrDatabaseClosed, fmt.Errorf("query: %w", orchestrator_dal.ErrDatabaseClosed)},
	}

	for _, tc := range cases {
		require.True(t, errors.Is(tc.wrapper, tc.base), "wrapped error should match base via errors.Is")
	}
}
