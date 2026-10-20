CREATE TABLE IF NOT EXISTS clicks (
    id          BIGSERIAL   PRIMARY KEY,
    code        VARCHAR(16) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    referrer    TEXT,
    user_agent  TEXT
);

CREATE INDEX IF NOT EXISTS clicks_code_time_idx ON clicks (code, occurred_at DESC);