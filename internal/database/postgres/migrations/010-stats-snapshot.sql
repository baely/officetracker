-- Stats snapshots for the public dashboard. Each row is a full, self-contained
-- snapshot of all widgets computed by the daily collector job, stored as JSON so
-- that new widgets can be added without a schema change. The dashboard reads the
-- most recent row.
CREATE TABLE IF NOT EXISTS "stats_snapshots" (
    "id"          SERIAL PRIMARY KEY,
    "computed_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
    "widgets"     JSONB NOT NULL
);

CREATE INDEX ON "stats_snapshots" ("computed_at" DESC);
