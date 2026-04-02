-- piko.name: CountDistinct
-- piko.command: one
SELECT COUNT(*) AS cnt FROM (SELECT DISTINCT col FROM foo) sub;
