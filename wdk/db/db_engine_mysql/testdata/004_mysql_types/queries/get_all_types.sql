-- piko.name: GetAllTypes
-- piko.command: one
SELECT
    id, tiny_col, small_col, medium_col, int_col, big_col,
    float_col, double_col, decimal_col,
    char_col, varchar_col, tinytext_col, text_col, mediumtext_col, longtext_col,
    binary_col, varbinary_col, tinyblob_col, blob_col, mediumblob_col, longblob_col,
    date_col, time_col, datetime_col, timestamp_col, year_col,
    json_col, bool_col
FROM type_test
WHERE id = ?;
