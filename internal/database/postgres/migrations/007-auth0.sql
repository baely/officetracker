CREATE TABLE IF NOT EXISTS "auth0_users" (
    "sub" TEXT,
    "profile" TEXT,
    "user_id" INTEGER REFERENCES "users" (user_id),
    PRIMARY KEY ("sub")
);
