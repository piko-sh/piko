CREATE TABLE alpha (
    id UInt64,
    label String
) ENGINE = MergeTree() ORDER BY id;

CREATE TABLE beta (
    id UInt64,
    label String
) ENGINE = MergeTree() ORDER BY id;

INSERT INTO alpha (id, label) VALUES (1, 'alpha-one'), (2, 'alpha-two');
INSERT INTO beta (id, label) VALUES (1, 'beta-one'), (2, 'beta-two');

EXCHANGE TABLES alpha AND beta;
