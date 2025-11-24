-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP,
    status TEXT DEFAULT 'active',
    notes TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL,
    drowsiness_score REAL NOT NULL,
    is_drowsy INTEGER DEFAULT 0,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_events_session;
DROP INDEX IF EXISTS idx_sessions_user;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd

ALTER TABLE sessions
ALTER COLUMN start_time TYPE TIMESTAMP WITH TIME ZONE
USING start_time AT TIME ZONE 'UTC';

ALTER TABLE sessions
ALTER COLUMN end_time TYPE TIMESTAMP WITH TIME ZONE
USING end_time AT TIME ZONE 'UTC';

ALTER TABLE events
ALTER COLUMN timestamp TYPE TIMESTAMP WITH TIME ZONE
USING timestamp AT TIME ZONE 'UTC';







