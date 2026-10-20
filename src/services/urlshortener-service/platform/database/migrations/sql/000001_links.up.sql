CREATE TABLE IF NOT EXISTS links (
    id         BIGSERIAL   PRIMARY KEY,
    code       VARCHAR(30) NOT NULL,
    target_url TEXT        NOT NULL,
    expires_at TIMESTAMPTZ,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS links_code_uidx ON links (code);