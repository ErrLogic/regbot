CREATE TABLE platform_accounts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform            TEXT NOT NULL CHECK (platform IN ('instagram','tiktok')),
    email               TEXT NOT NULL,
    username            TEXT NOT NULL DEFAULT '',
    encrypted_password  BYTEA NOT NULL,
    encryption_nonce    BYTEA NOT NULL,
    status              TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','locked','disabled')),
    device_serial       TEXT DEFAULT '',
    job_id              UUID,
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(platform, email)
);
