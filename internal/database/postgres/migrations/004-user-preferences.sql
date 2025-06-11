CREATE TABLE IF NOT EXISTS "user_preferences" (
    "user_id" INTEGER,
    "theme" TEXT DEFAULT 'default',
    "weather_enabled" BOOLEAN DEFAULT FALSE,
    "time_based_enabled" BOOLEAN DEFAULT FALSE,
    "location" TEXT DEFAULT NULL,
    PRIMARY KEY ("user_id")
);

ALTER TABLE "user_preferences" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("user_id");

-- Insert default preferences for existing users
INSERT INTO "user_preferences" ("user_id")
SELECT "user_id" FROM "users"
ON CONFLICT DO NOTHING;