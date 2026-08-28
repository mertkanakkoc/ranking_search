CREATE TABLE contents (
    id            TEXT PRIMARY KEY,
    external_id   TEXT NOT NULL,
    provider      TEXT NOT NULL REFERENCES providers(name),
    title         TEXT NOT NULL,
    content_type  TEXT NOT NULL,
    views         BIGINT NOT NULL DEFAULT 0,
    likes         BIGINT NOT NULL DEFAULT 0,
    reading_time  DOUBLE PRECISION NOT NULL DEFAULT 0,
    reactions     BIGINT NOT NULL DEFAULT 0,
    published_at  TIMESTAMPTZ NOT NULL,
    tags          TEXT[] NOT NULL DEFAULT '{}',
    raw_metrics   BYTEA,
    final_score   DOUBLE PRECISION NOT NULL,
    fetched_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', title)) STORED,

    UNIQUE (provider, external_id)
);

CREATE INDEX idx_contents_final_score  ON contents (final_score DESC);
CREATE INDEX idx_contents_content_type ON contents (content_type);
CREATE INDEX idx_contents_search       ON contents USING GIN (search_vector);
