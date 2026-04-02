-- piko.name: GenerateID
-- piko.command: one
SELECT unique_rowid() AS new_id;

-- piko.name: GetEvent
-- piko.command: one
SELECT id, name, occurred_at FROM events WHERE id = $1;
