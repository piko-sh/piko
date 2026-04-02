CREATE VIEW user_post_counts (id, name, role, post_count) AS
SELECT u.id, u.name, u.role, COUNT(p.id) AS post_count
FROM users u
LEFT JOIN posts p ON p.user_id = u.id
GROUP BY u.id;
