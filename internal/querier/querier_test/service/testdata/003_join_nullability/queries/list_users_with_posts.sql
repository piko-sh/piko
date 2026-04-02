-- piko.name: ListUsersWithPosts
-- piko.command: many
SELECT u.name, p.title FROM users u LEFT JOIN posts p ON p.author_id = u.id
