-- piko.query(name: UseUnknownCallee, command: one)
SELECT calls_unknown() AS result;

-- piko.query(name: UseKnownPure, command: one)
SELECT calls_known_pure(10) AS result;
