-- piko.query(EventsByTag, many)
SELECT id, tag FROM events ARRAY JOIN tags AS tag WHERE tag = {filter_tag:String};
