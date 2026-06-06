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

package db_engine_clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestDDL_CreateMaterializedViewRefreshEvery(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE MATERIALIZED VIEW mv
		REFRESH EVERY 1 HOUR OFFSET 5 MINUTE
		ENGINE = MergeTree() ORDER BY id
		AS SELECT id FROM t
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "EVERY", mutation.EngineSpecific["MV_REFRESH_KIND"])
	assert.Contains(t, mutation.EngineSpecific["MV_REFRESH_INTERVAL"], "HOUR")
	assert.Contains(t, mutation.EngineSpecific["MV_REFRESH_OFFSET"], "MINUTE")
}

func TestDDL_CreateMaterializedViewRefreshAfter(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE MATERIALIZED VIEW mv
		REFRESH AFTER 1 DAY
		RANDOMIZE FOR 1 HOUR
		DEPENDS ON other_mv
		APPEND
		ENGINE = MergeTree() ORDER BY id
		AS SELECT id FROM t
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "AFTER", mutation.EngineSpecific["MV_REFRESH_KIND"])
	assert.Contains(t, mutation.EngineSpecific["MV_REFRESH_AFTER"], "DAY")
	assert.Contains(t, mutation.EngineSpecific["MV_REFRESH_RANDOMIZE"], "HOUR")
	assert.Equal(t, "other_mv", mutation.EngineSpecific["MV_REFRESH_DEPENDS_ON"])
	assert.Equal(t, "true", mutation.EngineSpecific["MV_REFRESH_APPEND"])
}

func TestDDL_CreateMaterializedViewEmptyAndSecurity(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE MATERIALIZED VIEW mv
		ENGINE = MergeTree() ORDER BY id
		EMPTY
		DEFINER = admin
		SQL SECURITY DEFINER
		COMMENT 'a refresh view'
		AS SELECT id FROM t
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "true", mutation.EngineSpecific["MV_REFRESH_EMPTY"])
	assert.Equal(t, "admin", mutation.EngineSpecific["MV_DEFINER"])
	assert.Equal(t, "DEFINER", mutation.EngineSpecific["MV_SQL_SECURITY"])
	assert.Equal(t, "a refresh view", mutation.EngineSpecific["COMMENT"])
}

func TestDDL_CreateMaterializedViewToTarget(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE MATERIALIZED VIEW mv TO target_table
		AS SELECT id FROM t
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
	assert.Equal(t, "true", mutation.EngineSpecific["MATERIALIZED"])
	assert.Equal(t, "target_table", mutation.EngineSpecific["MATERIALIZED_TARGET"])
}

func TestDDL_CreateMaterializedViewToQualifiedTarget(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `
		CREATE MATERIALIZED VIEW analytics.mv TO analytics.target
		AS SELECT id FROM analytics.source
	`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "mv", mutation.TableName)
	assert.Equal(t, "analytics", mutation.SchemaName)
	assert.Equal(t, "target", mutation.EngineSpecific["MATERIALIZED_TARGET"])
}

func TestDDL_AlterAddProjection(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events ADD PROJECTION p_user (SELECT user_id, count() GROUP BY user_id)")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableAddProjection, mutation.Kind)
	assert.Equal(t, "p_user", mutation.EngineSpecific["PROJECTION_NAME"])
	assert.Contains(t, mutation.EngineSpecific["PROJECTION_SELECT"], "user_id")
}

func TestDDL_AlterDropProjection(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events DROP PROJECTION p_user")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableDropProjection, mutation.Kind)
	assert.Equal(t, "p_user", mutation.EngineSpecific["PROJECTION_NAME"])
}

func TestDDL_AlterMaterializeProjection(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events MATERIALIZE PROJECTION p_user IN PARTITION '2026-01'")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableMaterializeProjection, mutation.Kind)
	assert.Equal(t, "p_user", mutation.EngineSpecific["PROJECTION_NAME"])
	assert.Contains(t, mutation.EngineSpecific["MATERIALIZE_PARTITION"], "2026-01")
}

func TestDDL_AlterAddSkippingIndex(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events ADD INDEX idx col TYPE bloom_filter(0.01) GRANULARITY 2")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableAddSkippingIndex, mutation.Kind)
	assert.Equal(t, "idx", mutation.EngineSpecific["INDEX_NAME"])
	assert.Equal(t, "col", mutation.EngineSpecific["INDEX_EXPR"])
	assert.Contains(t, mutation.EngineSpecific["INDEX_TYPE"], "bloom_filter")
	assert.Equal(t, "2", mutation.EngineSpecific["INDEX_GRANULARITY"])
}

func TestDDL_AlterDropSkippingIndex(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events DROP INDEX idx")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableDropSkippingIndex, mutation.Kind)
	assert.Equal(t, "idx", mutation.EngineSpecific["INDEX_NAME"])
}

func TestDDL_AlterMaterializeIndex(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events MATERIALIZE INDEX idx")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableMaterializeIndex, mutation.Kind)
	assert.Equal(t, "idx", mutation.EngineSpecific["INDEX_NAME"])
}

func TestDDL_AlterAddConstraint(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events ADD CONSTRAINT chk_age CHECK age >= 0")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
	assert.Equal(t, "chk_age", mutation.EngineSpecific["CONSTRAINT_NAME"])
	assert.Contains(t, mutation.EngineSpecific["CONSTRAINT_CHECK"], "age")
}

func TestDDL_AlterDropConstraint(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events DROP CONSTRAINT chk_age")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableDropConstraint, mutation.Kind)
	assert.Equal(t, "chk_age", mutation.EngineSpecific["CONSTRAINT_NAME"])
}

func TestDDL_AlterAddStatistics(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events ADD STATISTICS a, b TYPE tdigest, uniq")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableAddStatistics, mutation.Kind)
	assert.Equal(t, "a,b", mutation.EngineSpecific["STATS_COLUMNS"])
	assert.Equal(t, "tdigest,uniq", mutation.EngineSpecific["STATS_TYPES"])
}

func TestDDL_AlterDropStatistics(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events DROP STATISTICS a, b")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableDropStatistics, mutation.Kind)
	assert.Equal(t, "a,b", mutation.EngineSpecific["STATS_COLUMNS"])
}

func TestDDL_AlterModifyStatistics(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events MODIFY STATISTICS a TYPE tdigest")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableModifyStatistics, mutation.Kind)
	assert.Equal(t, "a", mutation.EngineSpecific["STATS_COLUMNS"])
	assert.Equal(t, "tdigest", mutation.EngineSpecific["STATS_TYPES"])
}

func TestDDL_AlterMaterializeStatistics(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events MATERIALIZE STATISTICS a")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableMaterializeStatistics, mutation.Kind)
	assert.Equal(t, "a", mutation.EngineSpecific["STATS_COLUMNS"])
}

func TestDDL_AlterModifyQuery(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE mv MODIFY QUERY SELECT id, name FROM t")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableModifyQuery, mutation.Kind)
	assert.Contains(t, mutation.EngineSpecific["NEW_QUERY"], "SELECT")
	assert.Contains(t, mutation.EngineSpecific["NEW_QUERY"], "FROM")
}

func TestDDL_AlterModifyRefresh(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE mv MODIFY REFRESH EVERY 1 HOUR")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableModifyRefresh, mutation.Kind)
	assert.Equal(t, "REFRESH", mutation.EngineSpecific["MODIFY_REFRESH_KIND"])
	assert.Equal(t, "EVERY", mutation.EngineSpecific["MV_REFRESH_KIND"])
	assert.Contains(t, mutation.EngineSpecific["MV_REFRESH_INTERVAL"], "HOUR")
}

func TestDDL_AlterModifySQLSecurity(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE mv MODIFY SQL SECURITY INVOKER")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableModifyRefresh, mutation.Kind)
	assert.Equal(t, "SQL_SECURITY", mutation.EngineSpecific["MODIFY_REFRESH_KIND"])
	assert.Equal(t, "INVOKER", mutation.EngineSpecific["MV_SQL_SECURITY"])
}

func TestDDL_AlterModifyDefiner(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE mv MODIFY DEFINER = admin")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableModifyRefresh, mutation.Kind)
	assert.Equal(t, "DEFINER", mutation.EngineSpecific["MODIFY_REFRESH_KIND"])
	assert.Equal(t, "admin", mutation.EngineSpecific["MV_DEFINER"])
}

func TestDDL_AlterModifyColumnRemove(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events MODIFY COLUMN ts REMOVE DEFAULT")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableModifyColumn, mutation.Kind)
	assert.Equal(t, "ts", mutation.ColumnName)
	assert.Equal(t, "DEFAULT", mutation.EngineSpecific["COLUMN_REMOVE"])
}

func TestDDL_AlterModifyColumnModifyComment(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events MODIFY COLUMN ts MODIFY COMMENT 'updated timestamp'")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableModifyColumn, mutation.Kind)
	assert.Equal(t, "updated timestamp", mutation.EngineSpecific["COLUMN_MODIFY_COMMENT"])
}

func TestDDL_AlterTablePartitionFreeze(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events FREEZE PARTITION '2026-01' WITH NAME 'snap1'")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTablePartition, mutation.Kind)
	assert.Equal(t, "FREEZE", mutation.EngineSpecific["PARTITION_OP"])
	assert.Equal(t, "PARTITION", mutation.EngineSpecific["PARTITION_TARGET"])
	assert.Contains(t, mutation.EngineSpecific["PARTITION_EXPR"], "2026-01")
	assert.Equal(t, "snap1", mutation.EngineSpecific["PARTITION_BACKUP_NAME"])
}

func TestDDL_AlterTablePartitionMove(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events MOVE PARTITION '2026-01' TO DISK 'cold'")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTablePartition, mutation.Kind)
	assert.Equal(t, "MOVE", mutation.EngineSpecific["PARTITION_OP"])
	assert.Contains(t, mutation.EngineSpecific["PARTITION_DEST"], "DISK")
	assert.Contains(t, mutation.EngineSpecific["PARTITION_DEST"], "cold")
}

func TestDDL_AlterTablePartitionAttachAllFrom(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events ATTACH PARTITION ALL FROM source_table")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTablePartition, mutation.Kind)
	assert.Equal(t, "ATTACH", mutation.EngineSpecific["PARTITION_OP"])
	assert.Equal(t, "ALL", mutation.EngineSpecific["PARTITION_EXPR"])
	assert.Equal(t, "source_table", mutation.EngineSpecific["PARTITION_FROM_TABLE"])
}

func TestDDL_AlterTablePartitionDropDetached(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events DROP DETACHED PARTITION '2026-01'")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTablePartition, mutation.Kind)
	assert.Equal(t, "DROP", mutation.EngineSpecific["PARTITION_OP"])
	assert.Equal(t, "true", mutation.EngineSpecific["PARTITION_DETACHED"])
}

func TestDDL_AlterTablePartitionDetachPart(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "ALTER TABLE events DETACH PART '2026-01_1_1_0'")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTablePartition, mutation.Kind)
	assert.Equal(t, "DETACH", mutation.EngineSpecific["PARTITION_OP"])
	assert.Equal(t, "PART", mutation.EngineSpecific["PARTITION_TARGET"])
}

func TestDDL_CreateUserDropUser(t *testing.T) {
	t.Parallel()

	createMutation, err := applyDDL(t, "CREATE USER alice IDENTIFIED BY 'secret'")
	require.NoError(t, err)
	require.NotNil(t, createMutation)
	assert.Equal(t, querier_dto.MutationCreateUser, createMutation.Kind)
	assert.Equal(t, "alice", createMutation.EngineSpecific["USER_NAME"])

	dropMutation, err := applyDDL(t, "DROP USER alice")
	require.NoError(t, err)
	require.NotNil(t, dropMutation)
	assert.Equal(t, querier_dto.MutationDropUser, dropMutation.Kind)
	assert.Equal(t, "alice", dropMutation.EngineSpecific["USER_NAME"])
}

func TestDDL_CreateRoleAlterRoleDropRole(t *testing.T) {
	t.Parallel()

	createMutation, err := applyDDL(t, "CREATE ROLE admin")
	require.NoError(t, err)
	require.NotNil(t, createMutation)
	assert.Equal(t, querier_dto.MutationCreateRole, createMutation.Kind)
	assert.Equal(t, "admin", createMutation.EngineSpecific["ROLE_NAME"])

	alterMutation, err := applyDDL(t, "ALTER ROLE admin SETTINGS max_memory_usage = 1000")
	require.NoError(t, err)
	require.NotNil(t, alterMutation)
	assert.Equal(t, querier_dto.MutationAlterRole, alterMutation.Kind)

	dropMutation, err := applyDDL(t, "DROP ROLE admin")
	require.NoError(t, err)
	require.NotNil(t, dropMutation)
	assert.Equal(t, querier_dto.MutationDropRole, dropMutation.Kind)
}

func TestDDL_CreateRowPolicy(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE ROW POLICY p1 ON users USING id = 1 TO alice")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreatePolicy, mutation.Kind)
	assert.Equal(t, "p1", mutation.EngineSpecific["POLICY_NAME"])
}

func TestDDL_CreateQuota(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE QUOTA q1 FOR INTERVAL 1 day MAX queries = 100")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateQuota, mutation.Kind)
	assert.Equal(t, "q1", mutation.EngineSpecific["QUOTA_NAME"])
}

func TestDDL_CreateSettingsProfile(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, "CREATE SETTINGS PROFILE p1 SETTINGS max_memory_usage = 1000")
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateSettingsProfile, mutation.Kind)
	assert.Equal(t, "p1", mutation.EngineSpecific["PROFILE_NAME"])
}

func TestDDL_GrantRevoke(t *testing.T) {
	t.Parallel()

	grantMutation, err := applyDDL(t, "GRANT SELECT ON db.t TO alice")
	require.NoError(t, err)
	require.NotNil(t, grantMutation)
	assert.Equal(t, querier_dto.MutationGrantManagement, grantMutation.Kind)
	assert.Equal(t, "GRANT", grantMutation.EngineSpecific["RBAC_KIND"])
	assert.Contains(t, grantMutation.EngineSpecific["STATEMENT_BODY"], "SELECT")

	revokeMutation, err := applyDDL(t, "REVOKE SELECT ON db.t FROM alice")
	require.NoError(t, err)
	require.NotNil(t, revokeMutation)
	assert.Equal(t, querier_dto.MutationGrantManagement, revokeMutation.Kind)
	assert.Equal(t, "REVOKE", revokeMutation.EngineSpecific["RBAC_KIND"])
}

func TestDDL_ExplainDescribeCheckTableReadOnly(t *testing.T) {
	t.Parallel()

	for _, sql := range []string{
		"EXPLAIN PLAN SELECT 1",
		"DESCRIBE TABLE users",
		"DESC users",
		"CHECK TABLE users",
		"CHECK TABLE users PARTITION '2026-01'",
	} {
		t.Run(sql, func(t *testing.T) {
			t.Parallel()
			mutation, err := applyDDL(t, sql)
			require.NoError(t, err)
			assert.Nil(t, mutation, "expected read-only catalogue no-op")
		})
	}
}

func TestDDL_BackupRestore(t *testing.T) {
	t.Parallel()

	backupMutation, err := applyDDL(t, "BACKUP TABLE users TO Disk('archive', 'snap1')")
	require.NoError(t, err)
	require.NotNil(t, backupMutation)
	assert.Equal(t, querier_dto.MutationBackup, backupMutation.Kind)
	assert.Contains(t, backupMutation.EngineSpecific["STATEMENT_BODY"], "users")

	restoreMutation, err := applyDDL(t, "RESTORE TABLE users FROM Disk('archive', 'snap1')")
	require.NoError(t, err)
	require.NotNil(t, restoreMutation)
	assert.Equal(t, querier_dto.MutationRestore, restoreMutation.Kind)
}

func TestDDL_KillQueryKillMutation(t *testing.T) {
	t.Parallel()

	killQueryMutation, err := applyDDL(t, "KILL QUERY WHERE query_id = 'abc' SYNC")
	require.NoError(t, err)
	require.NotNil(t, killQueryMutation)
	assert.Equal(t, querier_dto.MutationKillQuery, killQueryMutation.Kind)

	killMutationMutation, err := applyDDL(t, "KILL MUTATION WHERE mutation_id = 'abc'")
	require.NoError(t, err)
	require.NotNil(t, killMutationMutation)
	assert.Equal(t, querier_dto.MutationKillMutation, killMutationMutation.Kind)
}

func TestDDL_AttachDetachTable(t *testing.T) {
	t.Parallel()

	attachMutation, err := applyDDL(t, "ATTACH TABLE users")
	require.NoError(t, err)
	require.NotNil(t, attachMutation)
	assert.Equal(t, querier_dto.MutationAttachTable, attachMutation.Kind)
	assert.Equal(t, "users", attachMutation.TableName)

	detachMutation, err := applyDDL(t, "DETACH TABLE users PERMANENTLY SYNC")
	require.NoError(t, err)
	require.NotNil(t, detachMutation)
	assert.Equal(t, querier_dto.MutationDetachTable, detachMutation.Kind)
	assert.Equal(t, "users", detachMutation.TableName)
}

func TestClassify_Explain(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindExplain, classify(t, "EXPLAIN SELECT 1"))
	assert.Equal(t, statementKindExplain, classify(t, "EXPLAIN PLAN SELECT id FROM t"))
}

func TestClassify_Describe(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindDescribeTable, classify(t, "DESCRIBE TABLE users"))
	assert.Equal(t, statementKindDescribeTable, classify(t, "DESC users"))
}

func TestClassify_CheckTable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCheckTable, classify(t, "CHECK TABLE users"))
}

func TestClassify_BackupRestore(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindBackup, classify(t, "BACKUP TABLE users TO Disk('backup', 'archive')"))
	assert.Equal(t, statementKindRestore, classify(t, "RESTORE TABLE users FROM Disk('backup', 'archive')"))
}

func TestClassify_Kill(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindKillQuery, classify(t, "KILL QUERY WHERE query_id = 'abc'"))
	assert.Equal(t, statementKindKillMutation, classify(t, "KILL MUTATION WHERE mutation_id = 'abc'"))
}

func TestClassify_AttachDetach(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindAttachTable, classify(t, "ATTACH TABLE users"))
	assert.Equal(t, statementKindDetachTable, classify(t, "DETACH TABLE users"))
}

func TestClassify_RBACCreateAlterDrop(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindCreateUser, classify(t, "CREATE USER alice IDENTIFIED BY 'secret'"))
	assert.Equal(t, statementKindAlterUser, classify(t, "ALTER USER alice IDENTIFIED BY 'newsecret'"))
	assert.Equal(t, statementKindDropUser, classify(t, "DROP USER alice"))

	assert.Equal(t, statementKindCreateRole, classify(t, "CREATE ROLE admins"))
	assert.Equal(t, statementKindAlterRole, classify(t, "ALTER ROLE admins SETTINGS max_memory_usage = 1000"))
	assert.Equal(t, statementKindDropRole, classify(t, "DROP ROLE admins"))

	assert.Equal(t, statementKindCreateQuota, classify(t, "CREATE QUOTA q1 FOR INTERVAL 1 day MAX queries = 100"))
	assert.Equal(t, statementKindAlterQuota, classify(t, "ALTER QUOTA q1 FOR INTERVAL 1 day MAX queries = 200"))
	assert.Equal(t, statementKindDropQuota, classify(t, "DROP QUOTA q1"))

	assert.Equal(t, statementKindCreatePolicy, classify(t, "CREATE ROW POLICY policy1 ON users USING id = 1 TO alice"))
	assert.Equal(t, statementKindAlterPolicy, classify(t, "ALTER ROW POLICY policy1 ON users USING id = 2"))
	assert.Equal(t, statementKindDropPolicy, classify(t, "DROP ROW POLICY policy1 ON users"))

	assert.Equal(t, statementKindCreateSettingsProfile, classify(t,
		"CREATE SETTINGS PROFILE p1 SETTINGS max_memory_usage = 1000"))
	assert.Equal(t, statementKindAlterSettingsProfile, classify(t,
		"ALTER SETTINGS PROFILE p1 SETTINGS max_memory_usage = 2000"))
	assert.Equal(t, statementKindDropSettingsProfile, classify(t, "DROP SETTINGS PROFILE p1"))
}

func TestClassify_GrantRevoke(t *testing.T) {
	t.Parallel()

	assert.Equal(t, statementKindGrant, classify(t, "GRANT SELECT ON db.t TO alice"))
	assert.Equal(t, statementKindRevoke, classify(t, "REVOKE SELECT ON db.t FROM alice"))
}

func TestRBAC_CreateUserOrReplaceParsesName(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `CREATE USER OR REPLACE alice IDENTIFIED BY 'secret'`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "alice", mutation.EngineSpecific[engineKeyUserName])
}

func TestRBAC_CreateUserIfNotExistsStillParses(t *testing.T) {
	t.Parallel()

	mutation, err := applyDDL(t, `CREATE USER IF NOT EXISTS bob`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "bob", mutation.EngineSpecific[engineKeyUserName])
}
