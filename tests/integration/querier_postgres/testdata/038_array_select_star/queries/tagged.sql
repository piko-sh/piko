-- piko.query(name: GetTagged, command: one)
SELECT t.* FROM tagged t WHERE t.id = $1;
