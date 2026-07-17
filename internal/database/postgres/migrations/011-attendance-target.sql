-- Attendance target: a monthly target percentage on user_preferences.
-- 0 = no target set (targets are optional).
ALTER TABLE "user_preferences"
ADD COLUMN IF NOT EXISTS "default_target_percent" INTEGER NOT NULL DEFAULT 0;
