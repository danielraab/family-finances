-- 0025_recurring_transactions: recurring transaction templates recorded
-- against an account — the same content fields as a transaction entry
-- (category required, counterparty/location/tags optional), plus a
-- recurrence rule (interval_unit/interval_count — "every interval_count
-- interval_units", replacing a growing list of named frequencies),
-- starts_on, and an optional ends_on (e.g. a cancelled subscription).
-- per_year_amount, ended, and next_suggested_date are never stored —
-- always computed server-side (see internal/recurringtransaction).
--
-- entries.recurring_transaction_id links an entry to the template it was
-- created from or has been linked to; a template with any non-deleted
-- linked entry cannot be deleted (enforced in application code, mirroring
-- categories' own in-use delete guard). See openspec change
-- `add-recurring-transactions`.

CREATE TABLE recurring_transactions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by     uuid NOT NULL REFERENCES users(id),
    account_id     uuid NOT NULL REFERENCES accounts(id),
    title          text NOT NULL,
    description    text,
    category_id    uuid NOT NULL REFERENCES categories(id),
    counterparty   text,
    location       text,
    amount         bigint NOT NULL,
    interval_unit  text NOT NULL CHECK (interval_unit IN ('day', 'week', 'month', 'year')),
    interval_count integer NOT NULL CHECK (interval_count > 0),
    starts_on      date NOT NULL,
    ends_on        date,
    deleted_at     timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_on IS NULL OR ends_on >= starts_on)
);

CREATE INDEX recurring_transactions_account_idx ON recurring_transactions (account_id);

CREATE TABLE recurring_transaction_tags (
    recurring_transaction_id uuid NOT NULL REFERENCES recurring_transactions(id) ON DELETE CASCADE,
    tag_id                   uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (recurring_transaction_id, tag_id)
);

ALTER TABLE entries ADD COLUMN recurring_transaction_id uuid REFERENCES recurring_transactions(id);
CREATE INDEX entries_recurring_transaction_idx ON entries (recurring_transaction_id) WHERE recurring_transaction_id IS NOT NULL;
