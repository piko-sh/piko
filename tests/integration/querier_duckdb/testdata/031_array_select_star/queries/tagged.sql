-- piko.query(name: GetTagged, command: one)
SELECT * FROM tagged WHERE id = $1;
