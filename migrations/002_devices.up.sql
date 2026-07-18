CREATE TABLE devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial          TEXT UNIQUE NOT NULL,
    model           TEXT DEFAULT '',
    state           TEXT NOT NULL DEFAULT 'offline'
                    CHECK (state IN ('offline','online','busy','unauthorized')),
    android_version TEXT DEFAULT '',
    last_seen_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
