-- Add schedule preferences columns to user_preferences table
ALTER TABLE "user_preferences" 
ADD COLUMN "schedule_monday" BOOLEAN DEFAULT FALSE,
ADD COLUMN "schedule_tuesday" BOOLEAN DEFAULT FALSE,
ADD COLUMN "schedule_wednesday" BOOLEAN DEFAULT FALSE,
ADD COLUMN "schedule_thursday" BOOLEAN DEFAULT FALSE,
ADD COLUMN "schedule_friday" BOOLEAN DEFAULT FALSE,
ADD COLUMN "schedule_saturday" BOOLEAN DEFAULT FALSE,
ADD COLUMN "schedule_sunday" BOOLEAN DEFAULT FALSE;