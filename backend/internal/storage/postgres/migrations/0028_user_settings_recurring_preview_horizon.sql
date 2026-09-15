-- 0028_user_settings_recurring_preview_horizon: adds the recurring preview
-- horizon preference (how far ahead an opted-in recurring-transaction
-- preview projects, default end_of_this_month when unset). See openspec
-- change `add-recurring-transaction-preview`.

ALTER TABLE user_settings
    ADD COLUMN recurring_preview_horizon text CHECK (
        recurring_preview_horizon IN (
            '1_month',
            '2_months',
            '3_months',
            'end_of_this_month',
            'end_of_next_month',
            'end_of_this_year'
        )
    );
