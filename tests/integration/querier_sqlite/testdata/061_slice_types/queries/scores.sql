-- piko.name: FetchByScores
-- piko.command: many
-- ?1 as piko.slice(score_values)
SELECT id, player, score FROM scores WHERE score IN (?1) ORDER BY id ASC;
