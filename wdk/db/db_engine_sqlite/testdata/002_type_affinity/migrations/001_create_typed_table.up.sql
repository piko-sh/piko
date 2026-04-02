CREATE TABLE typed_values (
  id INTEGER PRIMARY KEY,
  int_val INT NOT NULL,
  bigint_val BIGINT,
  text_val TEXT NOT NULL,
  varchar_val VARCHAR(255),
  real_val REAL,
  double_val DOUBLE,
  blob_val BLOB,
  numeric_val NUMERIC(10, 2),
  bool_val BOOLEAN,
  date_val DATE,
  datetime_val DATETIME,
  json_val JSON
);
