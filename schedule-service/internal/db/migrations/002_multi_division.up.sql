ALTER TABLE seasons DROP COLUMN division_id;

CREATE TABLE IF NOT EXISTS season_divisions (
    season_id   UUID NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    division_id TEXT NOT NULL,
    PRIMARY KEY (season_id, division_id)
);
