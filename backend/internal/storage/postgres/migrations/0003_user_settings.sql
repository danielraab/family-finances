-- 0003_user_settings: per-user preferences (display language, timezone,
-- default currency). One row per user, every column nullable — a missing
-- row and a NULL column are both "use the hardcoded application default,"
-- resolved in internal/settings. See openspec change `add-settings-page`.

CREATE TABLE user_settings (
    user_id          uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    language         text CHECK (language IN ('en', 'de')),
    timezone         text,
    default_currency char(3),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
