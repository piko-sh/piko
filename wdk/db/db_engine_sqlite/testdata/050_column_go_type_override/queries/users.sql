-- piko.query(GetUserUUID, one)
-- piko.column(id, go_type: "github.com/google/uuid.UUID")
SELECT id, email FROM users WHERE id = ?;
