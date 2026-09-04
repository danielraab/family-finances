-- 0004_user_administration: admin-facing user lifecycle. `disabled` is a
-- reversible lock-out; `deleted_at` is a one-way soft delete. Both are
-- enforced at sign-in and at session-resolution time (see internal/auth).
-- See openspec change `add-settings-page`.

ALTER TABLE users
    ADD COLUMN disabled   boolean NOT NULL DEFAULT false,
    ADD COLUMN deleted_at timestamptz;
