-- Update schedule preferences to use state integers instead of booleans
-- Add new columns for schedule states
ALTER TABLE "user_preferences" 
ADD COLUMN "schedule_monday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_tuesday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_wednesday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_thursday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_friday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_saturday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_sunday_state" INTEGER DEFAULT 0;

-- Migrate existing boolean data to state integers (true -> StateWorkFromOffice = 2)
UPDATE "user_preferences" SET
    schedule_monday_state = CASE WHEN schedule_monday = true THEN 2 ELSE 0 END,
    schedule_tuesday_state = CASE WHEN schedule_tuesday = true THEN 2 ELSE 0 END,
    schedule_wednesday_state = CASE WHEN schedule_wednesday = true THEN 2 ELSE 0 END,
    schedule_thursday_state = CASE WHEN schedule_thursday = true THEN 2 ELSE 0 END,
    schedule_friday_state = CASE WHEN schedule_friday = true THEN 2 ELSE 0 END,
    schedule_saturday_state = CASE WHEN schedule_saturday = true THEN 2 ELSE 0 END,
    schedule_sunday_state = CASE WHEN schedule_sunday = true THEN 2 ELSE 0 END;

-- Drop old boolean columns
ALTER TABLE "user_preferences" 
DROP COLUMN "schedule_monday",
DROP COLUMN "schedule_tuesday", 
DROP COLUMN "schedule_wednesday",
DROP COLUMN "schedule_thursday",
DROP COLUMN "schedule_friday",
DROP COLUMN "schedule_saturday",
DROP COLUMN "schedule_sunday";