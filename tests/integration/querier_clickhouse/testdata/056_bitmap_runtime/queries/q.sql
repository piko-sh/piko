-- piko.query(InsertRawVisitor, exec)
INSERT INTO raw_visitors (day, user_id) VALUES ({day:Date}, {user_id:UInt64});

-- piko.query(MaterialiseBitmap, exec)
INSERT INTO bitmap_visitors
SELECT day, groupBitmapState(user_id) AS visitors
FROM raw_visitors
GROUP BY day;

-- piko.query(UniqueVisitors, many)
SELECT
    day,
    bitmapCardinality(groupBitmapMergeState(visitors)) AS unique_count
FROM bitmap_visitors
GROUP BY day
ORDER BY day;
