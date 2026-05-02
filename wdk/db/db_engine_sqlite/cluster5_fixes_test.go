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

package db_engine_sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitStatements_TriggerBodyWithCaseExpression(t *testing.T) {
	t.Parallel()

	sql := `CREATE TRIGGER trg AFTER INSERT ON x BEGIN UPDATE y SET z = CASE WHEN NEW.a > 0 THEN 1 ELSE 0 END WHERE id = NEW.id; END;
SELECT 1;`

	tokens, err := tokenise(sql)
	require.NoError(t, err)
	statements := splitStatements(tokens)
	require.Len(t, statements, 2,
		"the trigger (with its inner CASE) and the trailing SELECT should be two statements")
}
