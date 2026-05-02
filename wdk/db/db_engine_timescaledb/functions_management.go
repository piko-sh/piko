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

// functions_management.go gathers the policy and hypertable management
// signature catalogue. The policy family (compression, retention,
// continuous aggregate, reorder, jobs) and the hypertable management
// family (chunks, tablespaces, hypercore, sizes) share parser
// machinery in parser_policy.go; collocating the catalogue
// registrations keeps the function-name list close to the parser
// dispatch table without growing the main functions.go beyond
// readable bounds.

package db_engine_timescaledb

import (
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

const (
	// continuousAggregatePolicyMinArguments is the count of required arguments for
	// add_continuous_aggregate_policy: continuous_aggregate, start_offset, end_offset, and
	// schedule_interval. The remaining arguments (if_not_exists, initial_start, timezone,
	// tiered-data and batching options) are optional.
	continuousAggregatePolicyMinArguments = 4
)

var (
	// dataModifyingManagementFunctions names TimescaleDB functions that mutate state.
	//
	// These cover the management, policy, and job functions. Every overload of each name is
	// marked DataAccessModifiesData after registration so a query projecting one (for
	// example SELECT drop_chunks(...)) is classified as data-modifying and routed to the
	// writer connection rather than a read replica. Read-only helpers (show_chunks, the
	// *_size and *_stats accessors) are deliberately excluded.
	dataModifyingManagementFunctions = []string{
		"add_compression_policy", "remove_compression_policy",
		"add_retention_policy", "remove_retention_policy",
		"add_continuous_aggregate_policy", "remove_continuous_aggregate_policy",
		"add_reorder_policy", "remove_reorder_policy",
		"add_job", "alter_job", "run_job", "delete_job",
		"create_hypertable", "add_dimension", "set_chunk_time_interval", "reorder_chunk",
		"compress_chunk", "decompress_chunk", "refresh_continuous_aggregate", "drop_chunks",
		"attach_tablespace", "detach_tablespace", "detach_tablespaces", "move_chunk",
		"merge_chunks", "merge_chunks_concurrently", "split_chunk",
		"attach_chunk", "detach_chunk", "create_chunk", "drop_chunk",
		"convert_to_columnstore", "convert_to_rowstore", "recompress_chunk", "rebuild_columnstore",
	}
)

// registerPolicyFamily registers the compression, retention, continuous-aggregate,
// reorder, and job-control policy management functions.
//
// The retention and continuous-aggregate policies accept extended option lists; add_job
// uses regproc for its first argument because TimescaleDB resolves the procedure name
// through the catalogue rather than treating it as opaque text. All scalar return shapes
// surface as job IDs or booleans except the alter_job record form which returns the
// captured job row.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerPolicyFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	registerCompressionPolicy(b)
	registerRetentionPolicy(b)
	registerContinuousAggregatePolicy(b)
	registerReorderPolicy(b)
	registerJobControlPolicy(b)
}

// registerCompressionPolicy registers the extended add_compression_policy signature plus
// remove_compression_policy.
//
// Only the hypertable and compress_after arguments are required (MinArguments = 2); the
// remaining scheduling and tiered-data options are optional, mirroring the sibling
// retention and continuous-aggregate policies. The compress_after argument is polymorphic
// in the upstream catalogue because it accepts either an interval or a concrete
// time-column value, so it is registered as Any to let the resolver bind either spelling.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerCompressionPolicy(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("add_compression_policy",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: "compress_after", Type: b.Any},
			db_engine_postgres.Arg{Name: "if_not_exists", Type: b.Boolean},
			db_engine_postgres.Arg{Name: paramNameScheduleInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameInitialStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
			db_engine_postgres.Arg{Name: "compress_created_before", Type: b.Interval},
		),
		b.Integer,
	).MinArguments = 2
	b.NeverNull("remove_compression_policy",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}),
		b.Boolean,
	)
}

// registerRetentionPolicy registers the extended add_retention_policy signature accepted
// by TimescaleDB 2.13 and later, plus the matching remove_retention_policy. The
// drop_after argument is polymorphic in the upstream catalogue because it can be either
// an interval or a concrete column type; registering it as Any lets the resolver bind
// either spelling.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerRetentionPolicy(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("add_retention_policy",
		b.Args(
			db_engine_postgres.Arg{Name: "relation", Type: regclassType()},
			db_engine_postgres.Arg{Name: "drop_after", Type: b.Any},
			db_engine_postgres.Arg{Name: paramNameScheduleInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameInitialStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
			db_engine_postgres.Arg{Name: "if_not_exists", Type: b.Boolean},
			db_engine_postgres.Arg{Name: "drop_created_before", Type: b.Interval},
		),
		b.Integer,
	).MinArguments = 2
	b.NeverNull("remove_retention_policy",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}),
		b.Boolean,
	)
}

// registerContinuousAggregatePolicy registers the extended
// add_continuous_aggregate_policy signature that adds tiered-data, batching, and
// refresh-order options on top of the original start/end offset + schedule shape. The
// start_offset and end_offset arguments are polymorphic because they accept both
// intervals and concrete time-column values.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerContinuousAggregatePolicy(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("add_continuous_aggregate_policy",
		b.Args(
			db_engine_postgres.Arg{Name: "continuous_aggregate", Type: regclassType()},
			db_engine_postgres.Arg{Name: "start_offset", Type: b.Any},
			db_engine_postgres.Arg{Name: "end_offset", Type: b.Any},
			db_engine_postgres.Arg{Name: paramNameScheduleInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: "if_not_exists", Type: b.Boolean},
			db_engine_postgres.Arg{Name: paramNameInitialStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
			db_engine_postgres.Arg{Name: "include_tiered_data", Type: b.Boolean},
			db_engine_postgres.Arg{Name: "buckets_per_batch", Type: b.Integer},
			db_engine_postgres.Arg{Name: "max_batches_per_execution", Type: b.Integer},
			db_engine_postgres.Arg{Name: "refresh_newest_first", Type: b.Boolean},
		),
		b.Integer,
	).MinArguments = continuousAggregatePolicyMinArguments
	b.NeverNull("remove_continuous_aggregate_policy",
		b.Args(db_engine_postgres.Arg{Name: "continuous_aggregate", Type: regclassType()}),
		b.Boolean,
	)
}

// registerReorderPolicy registers the reorder policy family. extension.go classifies
// add_reorder_policy / remove_reorder_policy as extension statement kinds; without these
// signatures the catalogue would refuse to resolve the function name even though the
// parser accepts the statement.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerReorderPolicy(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("add_reorder_policy",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: "index_name", Type: b.Text},
			db_engine_postgres.Arg{Name: "if_not_exists", Type: b.Boolean},
			db_engine_postgres.Arg{Name: paramNameInitialStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
		),
		b.Integer,
	).MinArguments = 2
	b.NeverNull("remove_reorder_policy",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}, db_engine_postgres.Arg{Name: "if_exists", Type: b.Boolean}),
		voidType(),
	).MinArguments = 1
}

// registerJobControlPolicy registers add_job / alter_job / run_job / delete_job.
//
// add_job takes a regproc procedure reference and a full set of scheduling options;
// alter_job mirrors the upstream record return so callers can read back the mutated job
// row. run_job and delete_job remain plain void operations on a job_id.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerJobControlPolicy(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("add_job",
		b.Args(
			db_engine_postgres.Arg{Name: "proc", Type: regprocType()},
			db_engine_postgres.Arg{Name: paramNameScheduleInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: "config", Type: b.JSONB},
			db_engine_postgres.Arg{Name: paramNameInitialStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "scheduled", Type: b.Boolean},
			db_engine_postgres.Arg{Name: "check_config", Type: regprocType()},
			db_engine_postgres.Arg{Name: "fixed_schedule", Type: b.Boolean},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
		),
		b.Integer,
	).MinArguments = 2
	b.NeverNull("alter_job",
		b.Args(
			db_engine_postgres.Arg{Name: "job_id", Type: b.Integer},
			db_engine_postgres.Arg{Name: paramNameScheduleInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: "max_runtime", Type: b.Interval},
			db_engine_postgres.Arg{Name: "max_retries", Type: b.Integer},
			db_engine_postgres.Arg{Name: "retry_period", Type: b.Interval},
			db_engine_postgres.Arg{Name: "scheduled", Type: b.Boolean},
			db_engine_postgres.Arg{Name: "config", Type: b.JSONB},
			db_engine_postgres.Arg{Name: "next_start", Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "if_exists", Type: b.Boolean},
			db_engine_postgres.Arg{Name: "check_config", Type: regprocType()},
			db_engine_postgres.Arg{Name: "fixed_schedule", Type: b.Boolean},
			db_engine_postgres.Arg{Name: paramNameInitialStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
		),
		opaqueType("alter_job_record"),
	).MinArguments = 1
	b.NeverNull("run_job", b.Args(db_engine_postgres.Arg{Name: "job_id", Type: b.Integer}), voidType())
	b.NeverNull("delete_job", b.Args(db_engine_postgres.Arg{Name: "job_id", Type: b.Integer}), voidType())
}

// registerHypertableManagementFamily registers the function-call DDL helpers users invoke
// from migrations: create_hypertable, add_dimension, set_chunk_time_interval, etc.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerHypertableManagementFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("create_hypertable",
		b.Args(db_engine_postgres.Arg{Name: "table", Type: regclassType()}, db_engine_postgres.Arg{Name: "time_column", Type: b.Text}),
		opaqueType("create_hypertable_record"),
	)
	b.NeverNull("create_hypertable",
		b.Args(
			db_engine_postgres.Arg{Name: "table", Type: regclassType()},
			db_engine_postgres.Arg{Name: "time_column", Type: b.Text},
			db_engine_postgres.Arg{Name: "partition_column", Type: b.Text},
			db_engine_postgres.Arg{Name: "number_partitions", Type: b.Integer},
		),
		opaqueType("create_hypertable_record"),
	).MinArguments = 2
	b.NeverNull("add_dimension",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: "column", Type: b.Text},
			db_engine_postgres.Arg{Name: "number_partitions", Type: b.Integer},
		),
		opaqueType("add_dimension_record"),
	)
	b.NeverNull("add_dimension",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: "column", Type: b.Text},
			db_engine_postgres.Arg{Name: "chunk_time_interval", Type: b.Interval},
		),
		opaqueType("add_dimension_record"),
	)
	b.NeverNull("set_chunk_time_interval",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}, db_engine_postgres.Arg{Name: "chunk_time_interval", Type: b.Interval}),
		voidType(),
	)
	b.NeverNull("reorder_chunk", b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}, db_engine_postgres.Arg{Name: "index", Type: regclassType()}), voidType()).MinArguments = 1
	registerChunkAccessOverloads(b)
	b.NeverNull("compress_chunk", b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}), regclassType())
	b.NeverNull("decompress_chunk", b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}), regclassType())
	b.NeverNull("refresh_continuous_aggregate",
		b.Args(
			db_engine_postgres.Arg{Name: "continuous_aggregate", Type: regclassType()},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "finish", Type: b.Timestamptz},
		),
		voidType(),
	)
	registerTablespaceManagement(b)
	registerChunkLifecycle(b)
	registerHypertableSizeManagement(b)
}

// registerChunkAccessOverloads registers the show_chunks and drop_chunks overloads.
//
// Each function exposes a base single-relation form plus a multi-argument named form that
// accepts older_than / newer_than / created_before / created_after time selectors. The
// upstream signatures use the `any` pseudotype for the time selectors because the column
// may be timestamptz, timestamp, date, or an integer epoch; we register them as Any so
// the resolver can bind any of those.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerChunkAccessOverloads(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addReturnsSet(b, "drop_chunks",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}, db_engine_postgres.Arg{Name: "older_than", Type: b.Timestamptz}),
		b.Text,
	)
	addReturnsSet(b, "drop_chunks",
		b.Args(
			db_engine_postgres.Arg{Name: "relation", Type: regclassType()},
			db_engine_postgres.Arg{Name: "older_than", Type: b.Any},
			db_engine_postgres.Arg{Name: "newer_than", Type: b.Any},
			db_engine_postgres.Arg{Name: "verbose", Type: b.Boolean},
			db_engine_postgres.Arg{Name: "created_before", Type: b.Any},
			db_engine_postgres.Arg{Name: "created_after", Type: b.Any},
		),
		b.Text,
	).MinArguments = 1
	addReturnsSet(b, "show_chunks", b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}), regclassType())
	addReturnsSet(b, "show_chunks",
		b.Args(
			db_engine_postgres.Arg{Name: "relation", Type: regclassType()},
			db_engine_postgres.Arg{Name: "older_than", Type: b.Any},
			db_engine_postgres.Arg{Name: "newer_than", Type: b.Any},
			db_engine_postgres.Arg{Name: "created_before", Type: b.Any},
			db_engine_postgres.Arg{Name: "created_after", Type: b.Any},
		),
		regclassType(),
	).MinArguments = 1
}

// registerTablespaceManagement registers attach/detach tablespace helpers.
//
// Tablespace names are the postgres `name` type, which the catalogue models as text (Piko
// does not distinguish `name` from `text` for Go type mapping), so they are registered as
// b.Text. The if_not_attached / if_attached options follow the established TimescaleDB
// convention of returning the integer count of affected tablespaces from
// detach_tablespaces.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerTablespaceManagement(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("attach_tablespace",
		b.Args(
			db_engine_postgres.Arg{Name: "tablespace", Type: b.Text},
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: "if_not_attached", Type: b.Boolean},
		),
		voidType(),
	).MinArguments = 2
	b.NeverNull("detach_tablespace",
		b.Args(
			db_engine_postgres.Arg{Name: "tablespace", Type: b.Text},
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: "if_attached", Type: b.Boolean},
		),
		b.Integer,
	).MinArguments = 2
	b.NeverNull("detach_tablespaces",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}),
		b.Integer,
	)
	b.NeverNull("move_chunk",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()},
			db_engine_postgres.Arg{Name: "destination_tablespace", Type: nameType()},
			db_engine_postgres.Arg{Name: "index_destination_tablespace", Type: nameType()},
			db_engine_postgres.Arg{Name: "reorder_index", Type: regclassType()},
			db_engine_postgres.Arg{Name: "verbose", Type: b.Boolean},
		),
		voidType(),
	).MinArguments = 2
}

// registerChunkLifecycle registers the chunk-level lifecycle helpers merge_chunks,
// merge_chunks_concurrently, split_chunk, attach_chunk, detach_chunk, create_chunk, and
// drop_chunk.
//
// The merge_chunks pair-form takes two chunks; the array-form takes a chunk array; both
// are exposed because upstream resolves either spelling. The variadic _concurrently form
// mirrors the array form with VARIADIC arity.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerChunkLifecycle(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("merge_chunks",
		b.Args(
			db_engine_postgres.Arg{Name: "chunk1", Type: regclassType()},
			db_engine_postgres.Arg{Name: "chunk2", Type: regclassType()},
			db_engine_postgres.Arg{Name: "concurrently", Type: b.Boolean},
		),
		voidType(),
	)
	b.NeverNull("merge_chunks",
		b.Args(db_engine_postgres.Arg{Name: "chunks", Type: arrayOf(regclassType())}, db_engine_postgres.Arg{Name: "concurrently", Type: b.Boolean}),
		voidType(),
	)
	b.Add("merge_chunks_concurrently", &querier_dto.FunctionSignature{
		Arguments:         b.Args(db_engine_postgres.Arg{Name: "chunks", Type: arrayOf(regclassType())}),
		ReturnType:        voidType(),
		IsVariadic:        true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	b.NeverNull("split_chunk",
		b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}, db_engine_postgres.Arg{Name: "split_at", Type: b.Timestamptz}),
		voidType(),
	)
	b.NeverNull("attach_chunk",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()},
			db_engine_postgres.Arg{Name: "slices", Type: b.JSONB},
		),
		voidType(),
	)
	b.NeverNull("detach_chunk",
		b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}),
		voidType(),
	)
	b.NeverNull("create_chunk",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()},
			db_engine_postgres.Arg{Name: "slices", Type: b.JSONB},
			db_engine_postgres.Arg{Name: "schema_name", Type: b.Text},
			db_engine_postgres.Arg{Name: "table_name", Type: b.Text},
			db_engine_postgres.Arg{Name: "chunk_table", Type: regclassType()},
		),
		opaqueType("create_chunk_record"),
	)
	b.NeverNull("drop_chunk",
		b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}),
		b.Boolean,
	)
}

// registerHypertableSizeManagement registers the approximate-size helpers that complement
// hypertable_size / hypertable_detailed_size. The approximate variants skip catalogue
// scans and are cheap enough to call from monitoring loops.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerHypertableSizeManagement(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("hypertable_approximate_size",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}),
		b.Bigint,
	)
	addReturnsSet(b, "hypertable_approximate_detailed_size",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}),
		opaqueType("hypertable_detailed_size_record"),
	)
}

// registerHypercoreFamily registers the hypercore (TS 2.18+) convert / recompress /
// rebuild procedures and the canonical hypertable_columnstore_stats /
// chunk_columnstore_stats aliases. The convert/recompress entries are procedures that
// return void; the stats overloads are SETOF helpers that share the column shape of
// chunk_compression_stats.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signatures.
func registerHypercoreFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("convert_to_columnstore",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()},
			db_engine_postgres.Arg{Name: "if_not_columnstore", Type: b.Boolean},
			db_engine_postgres.Arg{Name: "recompress", Type: b.Boolean},
		),
		voidType(),
	)
	b.NeverNull("convert_to_rowstore",
		b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}, db_engine_postgres.Arg{Name: "if_compressed", Type: b.Boolean}),
		voidType(),
	)
	b.NeverNull("recompress_chunk",
		b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}, db_engine_postgres.Arg{Name: "if_not_compressed", Type: b.Boolean}),
		voidType(),
	)
	b.NeverNull("rebuild_columnstore",
		b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}),
		voidType(),
	)
	addReturnsSet(b, "hypertable_columnstore_stats",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}),
		opaqueType(typeNameCompressionStatsRecord),
	)
	addReturnsSet(b, "chunk_columnstore_stats",
		b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}),
		opaqueType(typeNameCompressionStatsRecord),
	)
}

// markDataModifyingFunctions flags every overload of the management, policy, and job
// functions in dataModifyingManagementFunctions as DataAccessModifiesData, overriding the
// read-only default the catalogue builder applies. It runs after all signatures are
// registered so a query projecting one of these helpers is routed to the writer
// connection rather than a read replica.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) whose catalogue is updated in
// place.
func markDataModifyingFunctions(b *db_engine_postgres.FunctionCatalogueBuilder) {
	for _, name := range dataModifyingManagementFunctions {
		for _, signature := range b.Catalogue.Functions[name] {
			signature.DataAccess = querier_dto.DataAccessModifiesData
		}
	}
}
