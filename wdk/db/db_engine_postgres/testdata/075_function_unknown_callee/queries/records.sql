-- piko.name: UseUnknownCallee
-- piko.command: one
SELECT calls_unknown() AS result;

-- piko.name: UseKnownPure
-- piko.command: one
SELECT calls_known_pure(10) AS result;
