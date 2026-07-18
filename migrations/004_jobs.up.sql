CREATE TABLE jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type          TEXT NOT NULL CHECK (type IN (
                      'register','like','comment','update_profile','create_post','watch_live'
                  )),
    platform      TEXT NOT NULL CHECK (platform IN ('instagram','tiktok')),
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','running','completed','failed','cancelled')),
    priority      INT NOT NULL DEFAULT 0,
    params        JSONB NOT NULL DEFAULT '{}',
    result        JSONB,
    error_message TEXT,
    device_serial TEXT DEFAULT '',
    account_id    UUID REFERENCES platform_accounts(id),
    created_by    UUID REFERENCES users(id),
    retry_count   INT NOT NULL DEFAULT 0,
    max_retries   INT NOT NULL DEFAULT 3,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_pending ON jobs(status, priority DESC, created_at) WHERE status = 'pending';
