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

package querier_postgres

import (
	"context"
	sqldriver "database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
)

func TestDriver_CASWritesBindNullableFields(t *testing.T) {
	t.Parallel()

	fromStatuses := []string{"PROCESSING"}
	runNullableWriteCases(t, []nullableWriteCase{
		{name: "SettleTask last error", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, _, err := taskDriver.SettleTask(ctx, dalcore.TaskWriteParams{ID: "task-1", LastError: value})
			return err
		}},
		{name: "SettleTask result", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, _, err := taskDriver.SettleTask(ctx, dalcore.TaskWriteParams{ID: "task-1", Result: value})
			return err
		}},
		{name: "TransitionTask last error", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, _, err := taskDriver.TransitionTask(ctx, dalcore.TaskWriteParams{ID: "task-1", LastError: value}, fromStatuses)
			return err
		}},
		{name: "TransitionTask result", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, _, err := taskDriver.TransitionTask(ctx, dalcore.TaskWriteParams{ID: "task-1", Result: value}, fromStatuses)
			return err
		}},
	})
}

func TestDriver_RequestRerunBindsEveryFieldAsAValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		params dalcore.RerunParams
	}{
		{
			name:   "populated fields",
			params: dalcore.RerunParams{ID: "rerun-1", Payload: `{"a":1}`, Config: `{}`, DeduplicationKey: "key-1"},
		},
		{
			name:   "empty fields stay empty strings rather than NULL",
			params: dalcore.RerunParams{ID: "", Payload: "", Config: "", DeduplicationKey: ""},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			taskDriver, connector := newCapturingDriver(t)
			_, err := taskDriver.RequestRerun(t.Context(), testCase.params)
			require.NoError(t, err)

			statements := connector.recorded()
			require.Len(t, statements, 1)
			assert.Equal(t, []sqldriver.Value{
				testCase.params.ID, testCase.params.Payload, testCase.params.Config, testCase.params.DeduplicationKey,
			}, statements[0])
		})
	}
}
