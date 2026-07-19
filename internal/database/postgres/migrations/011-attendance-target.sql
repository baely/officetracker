-- Attendance target: a monthly target percentage on user_preferences.
-- Defaults to 50 (the common office mandate); 0 = no target.
ALTER TABLE "user_preferences"
ADD COLUMN IF NOT EXISTS "target_percent" INTEGER NOT NULL DEFAULT 50;
