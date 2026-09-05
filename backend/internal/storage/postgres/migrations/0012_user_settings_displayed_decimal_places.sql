-- 0012_user_settings_displayed_decimal_places: a per-user display-rounding
-- preference, separate from account-entries' fixed 4-decimal-place storage
-- — this only affects how a client renders an amount. See openspec change
-- `add-accounts-entries`.

ALTER TABLE user_settings
    ADD COLUMN displayed_decimal_places smallint
        CHECK (displayed_decimal_places BETWEEN 0 AND 4);
