-- Add intelligence source, target, and original event date fields to events.
-- Also update event_type enum: rename OTHER→UNCLASSIFIED, add HYBRID and MISINFORMATION.

ALTER TABLE fresnel.events
    ADD COLUMN IF NOT EXISTS intel_source TEXT NOT NULL DEFAULT 'Manual',
    ADD COLUMN IF NOT EXISTS target TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS original_event_date TIMESTAMPTZ;

-- Migrate any existing OTHER event types to UNCLASSIFIED
UPDATE fresnel.events SET event_type = 'UNCLASSIFIED' WHERE event_type = 'OTHER';
UPDATE fresnel.event_revisions SET event_type = 'UNCLASSIFIED' WHERE event_type = 'OTHER';
