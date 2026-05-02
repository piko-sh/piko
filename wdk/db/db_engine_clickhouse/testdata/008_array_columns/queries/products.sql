-- piko.query(GetProduct, one)
SELECT id, tags, scores, optional_codes FROM products WHERE id = {id:UInt64};
