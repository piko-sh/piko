-- piko.name: BareReadOnly
-- piko.command: many
-- piko.readonly
SELECT id, name FROM items;

-- piko.name: ExplicitReadOnlyTrue
-- piko.command: many
-- piko.readonly(true)
SELECT id, name FROM items;

-- piko.name: ExplicitReadOnlyFalse
-- piko.command: many
-- piko.readonly(false)
SELECT id, name FROM items;

-- piko.name: InsertOverriddenToReadOnly
-- piko.command: exec
-- piko.readonly
INSERT INTO items (id, name, quantity) VALUES ($1, $2, $3);
