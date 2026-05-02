-- piko.query(FullJoin, many)
SELECT l.id AS l_id, r.id AS r_id FROM left_t l FULL OUTER JOIN right_t r ON l.id = r.id;
