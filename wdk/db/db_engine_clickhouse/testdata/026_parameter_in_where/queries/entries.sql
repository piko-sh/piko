-- piko.query(FilterEntries, many)
SELECT id, category, score
FROM entries
WHERE category = {cat:String} AND score >= {min_score:Float64};
