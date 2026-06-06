-- piko.query(name: GetValueAsInteger, command: many)
SELECT id, value::integer AS int_value FROM data;

-- piko.query(name: GetAmountAsVarchar, command: many)
SELECT id, CAST(amount AS varchar) AS amount_text FROM data;
