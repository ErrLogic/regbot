CREATE TABLE job_logs (
    id          BIGSERIAL PRIMARY KEY,
    job_id      UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    level       TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug','info','warn','error')),
    step        TEXT DEFAULT '',
    message     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_job_logs_job ON job_logs(job_id);
CREATE INDEX idx_job_logs_time ON job_logs(job_id, created_at);
