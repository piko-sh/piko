-- piko.query(name: GetPlayerRankings, command: many)
SELECT player, game, points, row_number() OVER (PARTITION BY game ORDER BY points DESC) AS rank FROM scores;
