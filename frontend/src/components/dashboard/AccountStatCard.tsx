import type { DashboardCardConfig } from "../../lib/dashboardFilter";
import type { Account } from "../../lib/useAccountsWithBalances";
import { AccountCard } from "../AccountCard";
import { MissingReferenceCard } from "./MissingReferenceCard";

/**
 * An account_stat card: thin wrapper resolving config.account_id against
 * the caller's own accounts and delegating to the existing AccountCard
 * (title, institute, live balance, sign-colored, add-entry button gated
 * on append+ permission) — reused as-is rather than reimplemented, since
 * it already renders exactly what this card type needs.
 */
export function AccountStatCard({
  config,
  accounts,
  balances,
  displayedDecimalPlaces,
  locale,
}: {
  config: DashboardCardConfig;
  accounts: Account[];
  balances: Record<string, number>;
  displayedDecimalPlaces: number;
  locale: string;
}) {
  const account = accounts.find((a) => a.id === config.account_id);
  if (!account) {
    return <MissingReferenceCard />;
  }
  return (
    <AccountCard
      account={account}
      balance={balances[account.id]}
      displayedDecimalPlaces={displayedDecimalPlaces}
      locale={locale}
    />
  );
}
