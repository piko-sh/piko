-- piko.name: ReadOnlyWrapperPure
-- piko.command: one
SELECT wrapper_pure(42) AS result;

-- piko.name: ReadOnlyWrapperStable
-- piko.command: one
SELECT wrapper_dangerous(1) AS result;
