CREATE TABLE IF NOT EXISTS "auth0_users" (
    "sub" TEXT,
    "profile" TEXT,
    "user_id" INTEGER REFERENCES "users" (user_id),
    PRIMARY KEY ("sub")
);

INSERT INTO auth0_users (sub, profile, user_id)
SELECT
    'github|' || gh_id,
    json_build_object(
        'sub', 'github|' || gh_id,
        'nickname', gh_user,
        'picture', 'https://avatars.githubusercontent.com/u/' || gh_id || '?v=4'
    )::text,
    user_id
FROM gh_users
ON CONFLICT (sub) DO NOTHING;
