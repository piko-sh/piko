-- piko.query(GetUser, one)
-- piko.column(id, type: int8, go_type: "github.com/google/uuid.UUID")
SELECT id, email FROM users WHERE id = $1;
