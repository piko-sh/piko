-- piko.query(CombinedCounts, many)
WITH 
    today AS (SELECT count(*) AS c FROM events WHERE ts >= today()),
    week AS (SELECT count(*) AS c FROM events WHERE ts >= today() - 7)
SELECT (SELECT c FROM today) AS today_count, (SELECT c FROM week) AS week_count;
