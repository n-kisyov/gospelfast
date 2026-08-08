CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE versifications (
    id   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE books (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    versification_id  UUID NOT NULL REFERENCES versifications(id) ON DELETE CASCADE,
    name              VARCHAR(100) NOT NULL,
    short_name        VARCHAR(20) NOT NULL,
    testament         VARCHAR(3) NOT NULL,
    book_order        INT NOT NULL,
    chapter_count     INT NOT NULL,
    UNIQUE(versification_id, short_name),
    UNIQUE(versification_id, book_order)
);

CREATE TABLE translations (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_name        VARCHAR(20) NOT NULL UNIQUE,
    full_name         VARCHAR(255) NOT NULL,
    language          VARCHAR(10) NOT NULL DEFAULT 'en',
    versification_id  UUID NOT NULL REFERENCES versifications(id),
    description       TEXT,
    source_url        TEXT,
    copyright         TEXT,
    metadata          JSONB DEFAULT '{}',
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE verses (
    id              BIGSERIAL PRIMARY KEY,
    translation_id  UUID NOT NULL REFERENCES translations(id) ON DELETE CASCADE,
    book_id         UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    chapter         INT NOT NULL,
    verse           INT NOT NULL,
    text            TEXT NOT NULL,
    tsv             tsvector GENERATED ALWAYS AS (
                        to_tsvector('english', text)
                    ) STORED,
    UNIQUE(translation_id, book_id, chapter, verse)
);

CREATE INDEX idx_verses_tsv  ON verses USING GIN(tsv);
CREATE INDEX idx_verses_ref  ON verses(translation_id, book_id, chapter, verse);
CREATE INDEX idx_verses_trgm ON verses USING GIN(text gin_trgm_ops);
