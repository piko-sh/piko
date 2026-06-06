-- piko.query(name: FindSimilar, command: many)
SELECT id, vec_distance_L2(embedding, ?) AS distance FROM vectors ORDER BY distance LIMIT ?
