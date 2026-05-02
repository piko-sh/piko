-- piko.query(name: GenerateID, command: one)
SELECT unique_rowid() AS new_id;

-- piko.query(name: GetEvent, command: one)
SELECT id, name, occurred_at FROM events WHERE id = $1;
