-- piko.query(name: AggJSON, command: many)
SELECT category, json_agg(data) AS all_data, jsonb_agg(data) AS all_data_b
FROM events
GROUP BY category;
