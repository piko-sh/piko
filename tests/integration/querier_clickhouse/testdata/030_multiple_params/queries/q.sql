-- piko.query(Matching, many)
SELECT id, category, score FROM filtered WHERE category = {cat:String} AND score >= {min:Float64} ORDER BY id;
