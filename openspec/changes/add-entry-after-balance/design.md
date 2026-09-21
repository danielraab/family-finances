## Context

An entry list can be filtered and paginated, so the browser cannot derive a
correct running balance from the rows it happens to hold. The value must be
computed against the complete account ledger by the backend.

## Decisions

### Return the value on Entry

`after_balance` is a required, server-computed Entry property. This preserves
the entry summary's existing no-request-on-open behavior: every surface already
passes its listed Entry object into the modal.

### Follow canonical ledger ordering

PostgreSQL sums account-facing `entry_legs` through the represented entry's
`(booking_timestamp, id)` position. The in-memory test store uses its equivalent
`(booking_timestamp, insertion sequence)` ordering. This makes equal-timestamp
entries deterministic and means a self-transfer's listed receiving occurrence
gets the receiving account's balance rather than the sender's.

### Keep the balance-adjustment reading distinct

The existing nullable `balance` field is caller-supplied only for a balance
adjustment. The UI retains it under the clearer “Balance reading” label, while
`after_balance` is shown for every entry.

## Risks / Trade-offs

The PostgreSQL read uses a correlated aggregate for each returned Entry. It is
correct for filtered and paginated lists because it deliberately reads the full
ledger. Existing entry/account indexes support the correlation; query-plan
monitoring can motivate a later denormalization if real workloads require it.
