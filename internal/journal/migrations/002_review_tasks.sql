CREATE TABLE IF NOT EXISTS review_tasks (
    id TEXT PRIMARY KEY,
    journal_id TEXT NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    due_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    snapshot JSONB NOT NULL
);
