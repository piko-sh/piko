-- piko.name: RankedScores
-- piko.command: many
SELECT
    player_name,
    game,
    score,
    ROW_NUMBER() OVER (PARTITION BY game ORDER BY score DESC) AS rank_num,
    RANK() OVER (PARTITION BY game ORDER BY score DESC) AS rank_pos
FROM scores;
