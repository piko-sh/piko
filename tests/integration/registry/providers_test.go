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

//go:build integration

package registry_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	"piko.sh/piko/internal/registry/registry_domain"
)

func newOtterStore(t *testing.T) registry_domain.MetadataStore {
	t.Helper()
	dal, err := registry_otter.NewOtterDAL(registry_otter.Config{})
	require.NoError(t, err, "creating otter registry DAL")
	t.Cleanup(func() { _ = dal.Close() })
	return dal
}

func TestOtterConformance(t *testing.T) {
	t.Parallel()
	RunStoreSuite(t, Config{
		NewStore:                           newOtterStore,
		SupportsArtefactLocker:             false,
		SupportsRegistryInspector:          true,
		SupportsRollback:                   true,
		SupportsNestedTransactionRejection: true,
		SupportsGCHints:                    true,
		SupportsSRIHashPersistence:         true,
	})
}
