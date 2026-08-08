CREATE TABLE genbooks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    translation_id  UUID NOT NULL REFERENCES translations(id) ON DELETE CASCADE,
    path            TEXT NOT NULL,
    title           TEXT,
    content         TEXT NOT NULL,
    tsv             tsvector GENERATED ALWAYS AS (to_tsvector('english', content)) STORED,
    UNIQUE(translation_id, path)
);

CREATE INDEX idx_genbooks_path ON genbooks(translation_id, path);
CREATE INDEX idx_genbooks_tsv ON genbooks USING GIN(tsv);
