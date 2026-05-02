CREATE TABLE media_transformations (
    id UUID PRIMARY KEY,
    source_algorithm TEXT NOT NULL,
    priority INTEGER NOT NULL
);

CREATE TABLE source_media (
    id UUID PRIMARY KEY,
    algorithm TEXT NOT NULL
);
