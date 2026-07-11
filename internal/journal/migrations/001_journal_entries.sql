CREATE TABLE IF NOT EXISTS journal_entries (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    matrix_id TEXT NOT NULL,
    selected_option_id TEXT NOT NULL,
    snapshot JSONB NOT NULL
);
