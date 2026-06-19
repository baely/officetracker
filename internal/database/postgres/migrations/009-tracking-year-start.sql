-- Add the configurable tracking year start month to user_preferences.
-- 10 = October, matching the original hardcoded behaviour.
ALTER TABLE "user_preferences"
ADD COLUMN IF NOT EXISTS "tracking_year_start_month" INTEGER NOT NULL DEFAULT 10;
