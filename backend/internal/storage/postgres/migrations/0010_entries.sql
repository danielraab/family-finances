-- 0010_entries: transactions and balance adjustments recorded against an
-- account. kind/category_id CHECK: a transaction requires a category, a
-- balance adjustment may omit one. The (account_id, booking_timestamp, id)
-- index backs both the listing keyset and the live balance computation —
-- id (bigserial, insertion order) is the tie-break for entries sharing an
-- identical booking_timestamp. See openspec change `add-accounts-entries`.

CREATE TABLE entries (
    id                bigserial PRIMARY KEY,
    owner_id          uuid NOT NULL REFERENCES users(id),
    account_id        uuid NOT NULL REFERENCES accounts(id),
    kind              text NOT NULL CHECK (kind IN ('transaction', 'balance_adjustment')),
    amount            bigint NOT NULL,
    booking_timestamp timestamptz NOT NULL,
    title             text NOT NULL,
    description       text,
    category_id       uuid REFERENCES categories(id),
    deleted_at        timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (kind = 'transaction' AND category_id IS NOT NULL)
        OR (kind = 'balance_adjustment')
    )
);

CREATE INDEX entries_account_booking_idx ON entries (account_id, booking_timestamp, id);

CREATE TABLE entry_tags (
    entry_id bigint NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    tag_id   uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, tag_id)
);
