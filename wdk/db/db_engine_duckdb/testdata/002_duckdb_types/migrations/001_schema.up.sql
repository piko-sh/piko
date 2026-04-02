CREATE TABLE duckdb_types (
    id INTEGER PRIMARY KEY,
    tiny_val TINYINT,
    small_val SMALLINT,
    big_val BIGINT,
    huge_val HUGEINT,
    utiny_val UTINYINT,
    usmall_val USMALLINT,
    uint_val UINTEGER,
    ubig_val UBIGINT,
    blob_val BLOB,
    ts_s TIMESTAMP_S,
    ts_ms TIMESTAMP_MS,
    ts_ns TIMESTAMP_NS
);
