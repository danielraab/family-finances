-- 0007_accounts: accounts, each owned by exactly one user and visible only
-- to them. See openspec change `add-accounts-entries`.

CREATE TABLE accounts (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id            uuid NOT NULL REFERENCES users(id),
    title               text NOT NULL,
    description         text,
    type_id             uuid NOT NULL REFERENCES account_types(id),
    currency            char(3) NOT NULL,
    financial_institute text,
    opening_date        date NOT NULL,
    closing_date        date,
    deleted_at          timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CHECK (closing_date IS NULL OR closing_date >= opening_date)
);

CREATE INDEX accounts_owner_id_idx ON accounts (owner_id);
