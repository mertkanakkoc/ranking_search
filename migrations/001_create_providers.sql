CREATE TABLE providers (
    id              SERIAL PRIMARY KEY,
    name            TEXT UNIQUE NOT NULL,
    format          TEXT NOT NULL,
    base_url        TEXT NOT NULL,
    rate_limit_rps  NUMERIC NOT NULL DEFAULT 5,
    timeout_ms      INT NOT NULL DEFAULT 10000,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    last_fetched_at TIMESTAMPTZ,
    last_status     TEXT,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO providers (name, format, base_url, rate_limit_rps, timeout_ms) VALUES
    ('provider1', 'json', 'https://raw.githubusercontent.com/WEG-Technology/mock/refs/heads/main/v2/provider1', 5, 10000),
    ('provider2', 'xml',  'https://raw.githubusercontent.com/WEG-Technology/mock/refs/heads/main/v2/provider2', 5, 10000);
