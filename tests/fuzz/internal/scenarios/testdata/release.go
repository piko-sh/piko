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

package release_test

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	sqlstore "piko.sh/piko/internal/registry/registry_dal/querier_sqlite"
	d "piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/piko/wdk/db/db_schema_registry_sqlite"
)

func TestGeneratedPikoScenario(t *testing.T) {
	ctx := t.Context()
	ds, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_DRAWS"))
	require.NoError(t, err)
	path := "file:" + filepath.Join(t.TempDir(), "registry.db") + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	open := func() *sql.DB {
		dbase, err := sql.Open("sqlite", path)
		require.NoError(t, err)
		dbase.SetMaxOpenConns(1)
		return dbase
	}
	database := open()
	defer func() { require.NoError(t, database.Close()) }()
	mig, err := db.NewMigrator(db_engine_sqlite.SQLite(), database, db_schema_registry_sqlite.Migrations, "migrations")
	require.NoError(t, err)
	_, err = mig.Up(ctx)
	require.NoError(t, err)
	store := sqlstore.New(database)
	type lease struct {
		state                       string
		first, published, heartbeat int64
	}
	var leases [3]lease
	var seeds [3][]*registry_dto.ArtefactMeta
	var digests [3]string
	key := func(slot int) string {
		if slot < 2 {
			return "images/shared"
		}
		return "images/third"
	}
	release := func(slot int) string { return fmt.Sprintf("release-%d", slot) }
	for slot := range seeds {
		k := key(slot)
		at := time.Unix(1, 0)
		source := registry_dto.Variant{
			VariantID: "source", StorageBackendID: "overlay", StorageKey: k, ContentHash: k,
			SizeBytes: 32, MimeType: "text/plain", Status: registry_dto.VariantStatusReady, CreatedAt: at,
		}
		seeds[slot] = []*registry_dto.ArtefactMeta{{
			ID: k, SourcePath: k, CreatedAt: at, UpdatedAt: at, ActualVariants: []registry_dto.Variant{source},
		}}
		digests[slot] = d.SeedDigest(seeds[slot])
	}
	steps := append([]byte{0, 6, 1, 2, 8, 4, 3, 5, 2, 14, 6}, ds...)
	for step, b := range steps {
		slot := int(b/7) % len(seeds)
		id := release(slot)
		m := leases[slot]
		now := time.Unix(1_700_000_000+int64(step)*7200, 0)
		switch b % 7 {
		case 0:
			won, err := store.(d.ReleasePublisher).ClaimRelease(ctx, id, digests[slot], now.Unix(), now.Unix())
			require.NoError(t, err)
			require.Equal(t, m.state == "", won, "PIKO_CHECK release-claim: first claimant")
			if won {
				m = lease{state: d.ReleaseStatePublishing, first: now.Unix(), heartbeat: now.Unix()}
			}
		case 1, 2:
			if b%7 == 1 && m.state == d.ReleaseStatePublishing {
				now = time.Unix(m.heartbeat, 0)
			}
			want := d.PublishOutcomePublished
			switch {
			case m.state == d.ReleaseStatePublished:
				want = d.PublishOutcomeAlreadyPublished
			case m.state == d.ReleaseStatePublishing && now.Unix() == m.heartbeat:
				want = d.PublishOutcomeInProgress
			default:
				m = lease{state: d.ReleaseStatePublished, first: now.Unix(), published: now.Unix(), heartbeat: now.Unix()}
			}
			got, err := d.PublishRelease(ctx, store, id, digests[slot], seeds[slot], now)
			require.NoError(t, err)
			require.Equal(t, want, got, "PIKO_CHECK release-publish: ownership/takeover")
		case 3, 4:
			if b%7 == 4 {
				now = time.Unix(m.heartbeat-60, 0)
			}
			require.NoError(t, d.HeartbeatRelease(ctx, store, id, now))
			if m.state != "" {
				m.heartbeat = max(m.heartbeat, now.Unix())
			}
		case 5:
			require.NoError(t, d.RetireRelease(ctx, store, id))
			m = lease{}
		case 6:
			require.NoError(t, database.Close())
			database = open()
			store = sqlstore.New(database)
		}
		leases[slot] = m
		for i, w := range leases {
			got, present, err := store.(d.ReleasePublisher).GetRelease(ctx, release(i))
			require.NoError(t, err)
			require.Equal(t, w.state != "", present, "PIKO_CHECK release-present: step %d", step)
			if present {
				require.Equal(t, lease{state: w.state, first: w.first, published: w.published, heartbeat: w.heartbeat}, lease{state: got.State, first: got.FirstSeenAt, published: got.PublishedAt, heartbeat: got.HeartbeatAt}, "PIKO_CHECK release-lease: slot %d step %d", i, step)
				require.Equal(t, digests[i], got.PublishDigest)
			}
		}
		for _, k := range []string{key(0), key(2)} {
			want := 0
			for i, w := range leases {
				if w.state == d.ReleaseStatePublished && key(i) == k {
					want++
				}
			}
			refs, err := store.GetBlobRefCount(ctx, k)
			require.NoError(t, err)
			require.Equal(t, want, refs, "PIKO_CHECK release-refs: step %d key %s", step, k)
			art, err := store.FindArtefactByVariantStorageKey(ctx, k)
			if want == 0 {
				require.ErrorIs(t, err, d.ErrArtefactNotFound, "PIKO_CHECK release-removed: %s", k)
				require.Nil(t, art)
			} else {
				require.NoError(t, err)
				require.Equal(t, k, art.ID, "PIKO_CHECK release-lookup: %s", k)
			}
		}
	}
}
