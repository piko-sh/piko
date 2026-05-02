-- piko.column(users.id, go_type: "github.com/google/uuid.UUID")
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR NOT NULL
);
