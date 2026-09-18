import type { components } from "../../api/schema";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

/**
 * What a summary modal needs from the page that opened it. Every host page
 * already holds these for its own labels, so a summary resolves an
 * account/category/tag name without a request of its own.
 *
 * An id missing from a map is not an error — it means the visitor can't
 * see that entity (the same "not shared" case the ledger already renders).
 */
export type SummaryLookups = {
  accountById: Map<string, Account>;
  categoryById: Map<string, Category>;
  tagById: Map<string, Tag>;
  userId: string | undefined;
  displayedDecimalPlaces: number;
};

/** The raw lists a host page holds, before they become lookup maps. */
export type SummaryLookupSources = {
  accounts: readonly Account[];
  categories: readonly Category[];
  tags: readonly Tag[];
  userId: string | undefined;
  displayedDecimalPlaces: number;
};
