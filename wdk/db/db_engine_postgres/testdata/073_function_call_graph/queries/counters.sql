-- piko.query(name: ReadOnlyWrapperPure, command: one)
SELECT wrapper_pure(42) AS result;

-- piko.query(name: ReadOnlyWrapperStable, command: one)
SELECT wrapper_dangerous(1) AS result;
