-- piko.query(name: BareReadOnly, command: many, readonly: true)
SELECT id, name FROM items;

-- piko.query(name: ExplicitReadOnlyTrue, command: many, readonly: true)
SELECT id, name FROM items;

-- piko.query(name: ExplicitReadOnlyFalse, command: many, readonly: false)
SELECT id, name FROM items;

-- piko.query(name: OverrideVolatileToReadOnly, command: one, readonly: true)
SELECT volatile_func(?) AS result;

-- piko.query(name: InsertOverriddenToReadOnly, command: exec, readonly: true)
INSERT INTO items (name, quantity) VALUES (?, ?);

-- piko.query(name: MigrationOverriddenReadOnly, command: one)
SELECT overridden_readonly_func(?) AS result;

-- piko.query(name: MigrationOverriddenNotReadOnly, command: one)
SELECT overridden_not_readonly_func(?) AS result;
