-- piko.query(name: GetFirstTag, command: one)
SELECT id, tags[1] AS first_tag FROM posts WHERE id = $1;
