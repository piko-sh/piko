-- piko.query(name: GetRecordNotNull, command: one, nullable: false)
SELECT id, value, optional_num FROM records WHERE id = $1 LIMIT 1;
