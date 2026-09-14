import type { components } from "../api/schema";
import { compact } from "./compact";
import { resolveEffectiveRange, type WeekStart } from "./dateRangePresets";

export type DashboardCard = components["schemas"]["DashboardCard"];
export type DashboardCardType = components["schemas"]["DashboardCardType"];
export type DashboardCardConfig = components["schemas"]["DashboardCardConfig"];
type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

/**
 * Reports whether every account_id/category_id/tag_id a card's config
 * references still resolves in the caller's own, freshly-fetched
 * accounts/categories/tags — the check backing the "no longer
 * accessible" placeholder (a revoked share or a soft delete since the
 * card was created never re-validates server-side, per design.md).
 */
export function cardReferencesResolve(
  config: DashboardCardConfig,
  accounts: Account[],
  categories: Category[],
  tags: Tag[],
): boolean {
  if (config.account_id && !accounts.some((a) => a.id === config.account_id))
    return false;
  if (
    config.category_id &&
    !categories.some((c) => c.id === config.category_id)
  )
    return false;
  if (config.tag_id && !tags.some((t) => t.id === config.tag_id)) return false;
  return true;
}

/** A short, static description of a card's own inline filter — accounts/
 * categories/tags are assumed already reference-checked (see
 * cardReferencesResolve) by the time this renders. */
export function describeCardFilter(
  config: DashboardCardConfig,
  accounts: Account[],
  categories: Category[],
  tags: Tag[],
  allLabel: string,
): string {
  const parts: string[] = [];
  if (config.account_id) {
    const account = accounts.find((a) => a.id === config.account_id);
    if (account) parts.push(account.title);
  }
  if (config.category_id) {
    const category = categories.find((c) => c.id === config.category_id);
    if (category) parts.push(category.name);
  }
  if (config.tag_id) {
    const tag = tags.find((t) => t.id === config.tag_id);
    if (tag) parts.push(`#${tag.name}`);
  }
  return parts.length > 0 ? parts.join(" · ") : allLabel;
}

/**
 * Maps a card's inline filter to /reports' own search-param shape, for
 * the "click a filter-bearing card's title to open /reports prefilled"
 * behavior. Only the fields /reports itself understands are carried over
 * — a bar_chart card's `unit`/`columns` have no equivalent there and are
 * dropped; /reports still requires an explicit "Generate report" click,
 * so this only prefills the filter controls, never the results.
 */
export function cardReportsSearch(config: DashboardCardConfig) {
  return compact({
    category_id: config.category_id,
    include_subcategories:
      config.include_subcategories === false ? false : undefined,
    tag_id: config.tag_id,
    account_id: config.account_id,
    range: config.range?.preset,
    from: config.range?.from,
    to: config.range?.to,
  });
}

function toRangeStart(date: string): string {
  return new Date(`${date}T00:00:00.000Z`).toISOString();
}
function toRangeEnd(date: string): string {
  return new Date(`${date}T23:59:59.999Z`).toISOString();
}

/**
 * Resolves a card's stored `range` (a preset key or explicit from/to,
 * mirroring /reports' own shape) to concrete RFC3339 bounds, at render
 * time — never persisted, so a relative preset like "this_month" stays
 * live across reloads instead of freezing the range the card was created
 * with. Mirrors reports.tsx's own toRangeStart/toRangeEnd.
 */
function resolveCardDateRange(
  range: DashboardCardConfig["range"],
  weekStart: WeekStart,
): { from?: string; to?: string } {
  const effective = resolveEffectiveRange(
    { range: range?.preset, from: range?.from, to: range?.to },
    weekStart,
    undefined,
    new Date(),
  );
  return compact({
    from: effective.from ? toRangeStart(effective.from) : undefined,
    to: effective.to ? toRangeEnd(effective.to) : undefined,
  });
}

/**
 * Builds the query object shared by a query_stat/entry_list card's
 * `GET /api/entries` and `GET /api/entries/summary` calls, and a
 * bar_chart card's `GET /api/entries/flow-summary` call (which ignores
 * the `from`/`to` it doesn't accept) — one place translating a card's
 * inline Config into backend filter params, mirroring
 * reports.tsx's buildEntriesQuery/buildSummaryQuery.
 */
export function buildCardFilterQuery(
  config: DashboardCardConfig,
  weekStart: WeekStart,
) {
  const { from, to } = resolveCardDateRange(config.range, weekStart);
  return compact({
    account_id: config.account_id ? [config.account_id] : undefined,
    category_id: config.category_id,
    category_mode:
      config.category_id && config.include_subcategories === false
        ? ("exact" as const)
        : undefined,
    tag_id: config.tag_id,
    from,
    to,
  });
}
