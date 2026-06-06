-- piko.query(InRange, many)
SELECT id FROM t WHERE ts BETWEEN {start:DateTime} AND {end:DateTime};
