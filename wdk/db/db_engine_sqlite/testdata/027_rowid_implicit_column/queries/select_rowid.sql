-- piko.name: SelectRowID
-- piko.command: many
SELECT rowid, name FROM items WHERE rowid > ?
