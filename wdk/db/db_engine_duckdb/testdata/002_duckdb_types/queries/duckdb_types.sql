-- piko.name: GetAllDuckDBTypes
-- piko.command: many
SELECT id, tiny_val, small_val, big_val, huge_val, utiny_val, usmall_val, uint_val, ubig_val, blob_val, ts_s, ts_ms, ts_ns FROM duckdb_types;
