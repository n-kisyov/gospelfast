ALTER TABLE translations ADD COLUMN IF NOT EXISTS module_type VARCHAR(20) DEFAULT 'bible';

CREATE TABLE commentary_entries (
    id              BIGSERIAL PRIMARY KEY,
    translation_id  UUID NOT NULL REFERENCES translations(id) ON DELETE CASCADE,
    book_id         UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    chapter         INT NOT NULL,
    verse           INT NOT NULL,
    content         TEXT NOT NULL,
    UNIQUE(translation_id, book_id, chapter, verse)
);

CREATE INDEX idx_commentary_ref ON commentary_entries(translation_id, book_id, chapter, verse);

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username        VARCHAR(50) NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            VARCHAR(10) NOT NULL DEFAULT 'reader',
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE sessions (
    token       VARCHAR(64) PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
