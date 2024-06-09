CREATE TABLE IF NOT EXISTS "users" (
    "user_id" SERIAL,
    "gh_id" TEXT,
    PRIMARY KEY ("user_id")
);

CREATE TABLE IF NOT EXISTS "entries" (
    "user_id" INTEGER,
    "day" INTEGER,
    "month" INTEGER,
    "year" INTEGER,
    "state" INTEGER,
    PRIMARY KEY ("user_id", "day", "month", "year")
);

CREATE TABLE IF NOT EXISTS "notes" (
    "user_id" INTEGER,
    "month" INTEGER,
    "year" INTEGER,
    "notes" TEXT,
    PRIMARY KEY ("user_id", "month", "year")
);

CREATE TABLE IF NOT EXISTS "secrets" (
    "user_id" INTEGER,
    "secret" TEXT,
    "active" BOOLEAN,
    PRIMARY KEY ("secret")
);

CREATE INDEX ON "users" ("gh_id");

CREATE INDEX ON "entries" ("user_id", "month", "year");

CREATE INDEX ON "secrets" ("user_id");

CREATE INDEX ON "secrets" ("user_id", "active");

ALTER TABLE "entries" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("user_id");

ALTER TABLE "notes" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("user_id");

ALTER TABLE "secrets" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("user_id");
