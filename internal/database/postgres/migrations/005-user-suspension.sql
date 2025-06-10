ALTER TABLE "users" ADD COLUMN "suspended" BOOLEAN DEFAULT FALSE;

CREATE INDEX ON "users" ("suspended");