CREATE EXTENSION IF NOT EXISTS vector;

SET search_path TO fresnel, public;

CREATE TABLE event_embeddings (
    event_id UUID PRIMARY KEY REFERENCES fresnel.events(id) ON DELETE CASCADE,
    embedding vector(768),
    model_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- IVFFLAT requires training data; add index when embeddings are populated (Phase 2).
