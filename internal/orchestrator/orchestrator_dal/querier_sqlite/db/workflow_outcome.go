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

package db

import (
	"context"
	"database/sql"
	"errors"
)

// WorkflowError reads persisted failure state for completion-event reconciliation.
func (queries *Queries) WorkflowError(ctx context.Context, workflowID string) (outcome error, readErr error) {
	var failure string
	err := queries.reader.QueryRowContext(ctx,
		`SELECT COALESCE(last_error, 'task failed') FROM tasks WHERE workflow_id = ?1 AND status = 'FAILED' ORDER BY id LIMIT 1`,
		workflowID).Scan(&failure)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return errors.New(failure), nil
}
