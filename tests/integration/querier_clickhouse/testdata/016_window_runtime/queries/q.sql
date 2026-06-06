-- piko.query(Ranked, many)
SELECT player, score, row_number() OVER (ORDER BY score DESC) AS rank FROM scores;
