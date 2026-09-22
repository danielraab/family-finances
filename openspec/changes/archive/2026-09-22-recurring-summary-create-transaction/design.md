# Design

## Which date the action books

The summary is opened from three places, and only one of them names an
occurrence:

| Opened from | Occurrence known? | What the action sends |
| --- | --- | --- |
| An Upcoming row's title | yes — that row's projected date | `recurring_transaction_id` + `booking_timestamp` |
| A `/recurring` list row | no | `recurring_transaction_id` only |
| A linked entry's badge | no (the entry already exists) | `recurring_transaction_id` only |

So the date is an optional prop threaded from the row that knows it, not
something the modal derives. It could not derive it: the modal fetches the
template, which carries `next_suggested_date` — the *first* unbooked
occurrence. For the first Upcoming row those two agree, and for every row
below it they do not. Booking the third row and getting the first row's
date is the kind of wrong that is easy to miss, because the form does
prefill *a* plausible date.

`booking_timestamp` is omitted rather than sent empty when unknown, so the
`/recurring` and badge paths produce byte-identical URLs to the ones they
produce today and the entry form's existing default applies untouched.

## Why a bordered action rather than a second filled one

`SummaryActions` already has two treatments: `summaryLinkClass` (filled,
for the one primary action) and `summaryQuietButtonClass` (unfilled, for
Close and Back — dismissals). Create transaction is neither: it is a real
action, so the quiet class would under-sell it and make it look like
another way out of the modal; but a second filled button beside Edit would
leave the reader with two equally loud primaries and no answer to which
one the modal is for.

A bordered `summaryOutlineLinkClass` sits between them, and it is the same
control `/recurring`'s rows already use for this exact action — so the
gesture a reader learned in the list looks the same in the summary.

## Why Edit stays the primary

Edit is the only action that operates on the thing the modal is showing.
Create transaction operates on a *new* entry derived from it, and it
navigates away to a form the reader then has to complete. Keeping Edit
filled also means this change adds an action without re-ranking the two
that were already there.
