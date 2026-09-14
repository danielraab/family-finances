import { Link } from "@tanstack/react-router";
import type { components } from "../../api/schema";
import {
  cardReportsSearch,
  type DashboardCardConfig,
  describeCardFilter,
} from "../../lib/dashboardFilter";
import type { Account } from "../../lib/useAccountsWithBalances";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

/**
 * The clickable heading shared by query_stat/entry_list/bar_chart cards
 * (never account_stat, whose heading is its account's own name and
 * already links to that account's details page): config.title when set,
 * falling back to the same generated filter summary describeCardFilter
 * already produces. Always a link through to /reports with this card's
 * account/category/tag/date-range filter prefilled into the controls —
 * /reports still requires its own "Generate report" click, so this only
 * prefills, never auto-runs, the report.
 */
export function CardTitleLink({
  config,
  accounts,
  categories,
  tags,
  allLabel,
}: {
  config: DashboardCardConfig;
  accounts: Account[];
  categories: Category[];
  tags: Tag[];
  allLabel: string;
}) {
  const title =
    config.title?.trim() ||
    describeCardFilter(config, accounts, categories, tags, allLabel);
  return (
    <Link
      to="/reports"
      search={cardReportsSearch(config)}
      className="truncate text-sm font-medium text-zinc-500 transition-colors hover:text-zinc-900 hover:underline dark:text-zinc-400 dark:hover:text-zinc-100"
    >
      {title}
    </Link>
  );
}
