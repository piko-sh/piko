-- piko.name: GroupedUserPosts
-- piko.command: many
SELECT u.name, u.email, p.title FROM users u FULL OUTER JOIN posts p ON p.author_id = u.id GROUP BY u.id, p.id
