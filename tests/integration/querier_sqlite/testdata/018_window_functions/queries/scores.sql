-- piko.name: ListRankedScores
-- piko.command: many
SELECT
    id,
    player,
    score,
    ROW_NUMBER() OVER (ORDER BY score DESC) AS rank_number,
    RANK() OVER (ORDER BY score DESC) AS rank_position,
    SUM(score) OVER (PARTITION BY player ORDER BY game_date) AS running_total
FROM scores
ORDER BY rank_number;
