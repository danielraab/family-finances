-- 0026_user_settings_week_start: adds the week_start preference (monday or
-- sunday, default monday when unset) used to anchor week-based date-range
-- presets on the client. See openspec change `add-date-range-presets`.

ALTER TABLE user_settings
    ADD COLUMN week_start text CHECK (week_start IN ('monday', 'sunday'));
