DROP USER IF EXISTS test_user;

DROP ROLE IF EXISTS test_reader;

CREATE USER test_user IDENTIFIED WITH no_password;

CREATE ROLE test_reader;

GRANT test_reader TO test_user;

CREATE TABLE noop (
    id UInt64
) ENGINE = MergeTree() ORDER BY id;
