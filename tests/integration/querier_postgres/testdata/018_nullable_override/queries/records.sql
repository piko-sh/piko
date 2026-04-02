-- piko.name: GetRecordNullable
-- piko.command: one
-- piko.nullable: true
SELECT id, value, optional_num FROM records WHERE id = $1;
