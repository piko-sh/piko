-- piko.query(name: GetTagged, command: one)
SELECT id, tags FROM tagged WHERE id = $1;
