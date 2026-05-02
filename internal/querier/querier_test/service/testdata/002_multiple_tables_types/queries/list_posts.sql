-- piko.query(name: ListPosts, command: many)
SELECT id, title, published FROM posts WHERE author_id = $1
