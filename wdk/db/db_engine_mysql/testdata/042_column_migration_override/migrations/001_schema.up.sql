-- piko.column(users.id, go_type: "github.com/google/uuid.UUID")
CREATE TABLE users (
    id BINARY(16) NOT NULL PRIMARY KEY,
    email VARCHAR(255) NOT NULL
);
