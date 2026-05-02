-- piko.query(InsertVisit, exec)
INSERT INTO source_visits (visit_day, visitor) VALUES ({visit_day:Date}, {visitor:String});

-- piko.query(MaterialiseStates, exec)
INSERT INTO daily_uniques
SELECT visit_day, uniqState(visitor) AS visitor_state
FROM source_visits
GROUP BY visit_day;

-- piko.query(MergedCounts, many)
SELECT visit_day, uniqMerge(visitor_state) AS unique_visitors
FROM daily_uniques
GROUP BY visit_day
ORDER BY visit_day;
