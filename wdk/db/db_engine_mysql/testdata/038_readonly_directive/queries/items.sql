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

-- piko.name: OverrideVolatileToReadOnly
-- piko.command: one
-- piko.readonly
SELECT volatile_func(?) AS result;

-- piko.name: InsertOverriddenToReadOnly
-- piko.command: exec
-- piko.readonly
INSERT INTO items (name, quantity) VALUES (?, ?);

-- piko.name: MigrationOverriddenReadOnly
-- piko.command: one
SELECT overridden_readonly_func(?) AS result;

-- piko.name: MigrationOverriddenNotReadOnly
-- piko.command: one
SELECT overridden_not_readonly_func(?) AS result;
