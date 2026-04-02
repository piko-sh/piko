-- piko.name: GetValueAsInteger
-- piko.command: many
SELECT id, value::integer AS int_value FROM data;

-- piko.name: GetAmountAsVarchar
-- piko.command: many
SELECT id, CAST(amount AS varchar) AS amount_text FROM data;
