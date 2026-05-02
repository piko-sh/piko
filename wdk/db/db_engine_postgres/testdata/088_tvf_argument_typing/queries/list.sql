-- piko.query(name: ListUsersSince, command: many)
SELECT user_id, user_email
FROM app.list_users_since($1, $2);
