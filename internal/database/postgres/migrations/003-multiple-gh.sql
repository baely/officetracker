CREATE TABLE IF NOT EXISTS "gh_users" (
    "gh_id" TEXT,
    "user_id" INTEGER,
    "gh_user" TEXT,
    PRIMARY KEY ("gh_id")
);

ALTER TABLE "gh_users" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("user_id");

INSERT INTO "gh_users" ("gh_id", "user_id", "gh_user") SELECT "gh_id", "user_id", "gh_user" FROM "users";