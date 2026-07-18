CREATE TABLE media (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename      TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size_bytes    BIGINT NOT NULL,
    data          BYTEA NOT NULL,
    uploaded_by   UUID REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_media_created ON media(created_at DESC);
