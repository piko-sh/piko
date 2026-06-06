-- piko.query(USRecords, many)
SELECT id, val FROM events PREWHERE region = 'US' WHERE val > 0;
