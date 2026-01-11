-- Add new columns to secrets table
ALTER TABLE "secrets" ADD COLUMN "token_id" SERIAL;
ALTER TABLE "secrets" ADD COLUMN "name" TEXT DEFAULT 'Developer API Token';
ALTER TABLE "secrets" ADD COLUMN "created_at" TIMESTAMP DEFAULT TO_TIMESTAMP(0);

-- Populate defaults for existing records
UPDATE "secrets" SET created_at = TO_TIMESTAMP(0) WHERE created_at IS NULL;
UPDATE "secrets" SET name = 'Developer API Token' WHERE name IS NULL;

-- Change primary key from secret to token_id
ALTER TABLE "secrets" DROP CONSTRAINT "secrets_pkey";
ALTER TABLE "secrets" ADD PRIMARY KEY ("token_id");

-- Add index on secret for auth lookups
CREATE INDEX ON "secrets" ("secret");
