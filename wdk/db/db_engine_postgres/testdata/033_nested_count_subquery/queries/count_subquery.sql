-- piko.query(name: CountDistinct, command: one)
SELECT COUNT(*) AS cnt FROM (SELECT DISTINCT col FROM foo) sub;
