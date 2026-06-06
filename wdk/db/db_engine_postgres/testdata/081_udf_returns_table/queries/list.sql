-- piko.query(name: ListUsers, command: many)
SELECT user_id, user_email
FROM app.list_users($1::integer);
