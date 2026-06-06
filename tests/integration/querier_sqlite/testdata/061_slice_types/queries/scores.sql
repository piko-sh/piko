-- piko.query(name: FetchByScores, command: many)
-- ?1 as piko.param(score_values, kind: slice)
SELECT id, player, score FROM scores WHERE score IN (?1) ORDER BY id ASC;
