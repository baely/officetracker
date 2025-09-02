-- Add schedule preferences columns to user_preferences table
-- Uses state integers: 0=Untracked, 1=WorkFromHome, 2=WorkFromOffice, 3=Other
ALTER TABLE "user_preferences" 
ADD COLUMN "schedule_monday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_tuesday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_wednesday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_thursday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_friday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_saturday_state" INTEGER DEFAULT 0,
ADD COLUMN "schedule_sunday_state" INTEGER DEFAULT 0;