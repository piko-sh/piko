-- piko.query(InsertRawView, exec)
INSERT INTO raw_views (page_id, visitor, views) VALUES ({page_id:UInt64}, {visitor:String}, {views:UInt64});

-- piko.query(MaterialiseMetrics, exec)
INSERT INTO page_metrics
SELECT
    page_id,
    uniqState(visitor) AS visitor_state,
    sumState(views)    AS view_total
FROM raw_views
GROUP BY page_id;

-- piko.query(MergedMetrics, many)
SELECT
    page_id,
    uniqMerge(visitor_state) AS unique_visitors,
    sumMerge(view_total)     AS total_views
FROM page_metrics
GROUP BY page_id
ORDER BY page_id;
