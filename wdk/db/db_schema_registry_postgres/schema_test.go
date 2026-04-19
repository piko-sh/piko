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

package db_schema_registry_postgres

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationsEmbedded(t *testing.T) {
	const up = "migrations/001_registry_metadata.up.sql"

	data, err := fs.ReadFile(Migrations, up)
	require.NoErrorf(t, err, "reading embedded %s: %v", up, err)
	require.NotEmptyf(t, data, "embedded %s is empty", up)
}
