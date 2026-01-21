CREATE TABLE todos (
    id                 UUID PRIMARY KEY,
    title              TEXT NOT NULL,
    status             TEXT NOT NULL,
    -- Email is only relevant once status = DONE
    email_status       TEXT NOT NULL, 
    email_attempts     INTEGER NOT NULL DEFAULT 0,
    email_last_error   TEXT,
    email_provider_id  TEXT,

    due_date           DATE NOT NULL,

    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX IF NOT EXISTS idx_todos_created_at_desc ON todos(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_todos_status_created_at ON todos(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_todos_status_email_status_created_at ON todos(status, email_status, created_at DESC);
