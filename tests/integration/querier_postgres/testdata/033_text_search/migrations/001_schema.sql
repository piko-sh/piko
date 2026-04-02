CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('english', title || ' ' || body)
    ) STORED
);

CREATE INDEX idx_articles_search ON articles USING GIN (search_vector);
