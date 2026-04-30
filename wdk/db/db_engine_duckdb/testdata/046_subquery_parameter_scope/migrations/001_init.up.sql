CREATE TABLE profiles (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    role VARCHAR NOT NULL
);

CREATE TABLE user_profiles (
    user_id INTEGER NOT NULL,
    profile_id INTEGER NOT NULL,
    PRIMARY KEY (user_id, profile_id)
);
