-- piko.column(users.id, go_type: "github.com/google/uuid.UUID")
CREATE TABLE users (
    id BLOB PRIMARY KEY,
    email TEXT NOT NULL
);
