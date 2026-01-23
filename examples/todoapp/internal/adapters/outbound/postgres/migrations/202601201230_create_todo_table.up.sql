CREATE TABLE todos (
    id                 UUID PRIMARY KEY,
    title              TEXT NOT NULL,
    status             TEXT NOT NULL,
    due_date           DATE NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX IF NOT EXISTS idx_todos_created_at_desc ON todos(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_todos_status_created_at ON todos(status, created_at DESC);


CREATE TABLE board_summary (
    id UUID PRIMARY KEY,
    summary JSONB NOT NULL,
    model TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    source_version BIGINT NOT NULL
);


CREATE TABLE outbox_events (
    id                 UUID PRIMARY KEY,
    entity_type      TEXT NOT NULL,
    entity_id        UUID NOT NULL,
    topic            TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    payload            JSONB NOT NULL,
    status             TEXT NOT NULL DEFAULT 'PENDING',
    retry_count        INTEGER NOT NULL DEFAULT 0,
    max_retries        INTEGER NOT NULL DEFAULT 3,
    last_error         TEXT,
    created_at         TIMESTAMPTZ NOT NULL
);

-- Index for unprocessed events (ordered by creation time for FIFO processing)
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(status, created_at ASC);

