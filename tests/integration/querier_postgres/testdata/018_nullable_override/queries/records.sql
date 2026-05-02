-- piko.query(name: GetRecordNullable, command: one, nullable: true)
SELECT id, value, optional_num FROM records WHERE id = $1;
