-- 0006_account_types: a flat, instance-wide, admin-managed lookup of account
-- types (e.g. checking, savings, credit card). See openspec change
-- `add-accounts-entries`.

CREATE TABLE account_types (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);
