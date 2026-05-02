-- piko.query(InsertScore, exec)
INSERT INTO scores (id, player, score) VALUES ({id:UInt64}, {player:String}, {score:UInt32});

-- piko.query(TopScorePerPlayer, many)
SELECT player, score
FROM scores
QUALIFY row_number() OVER (PARTITION BY player ORDER BY score DESC) = 1
ORDER BY player;
