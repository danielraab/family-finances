import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { components } from "../api/schema";

export type Account = components["schemas"]["Account"];
export type AccountType = components["schemas"]["AccountType"];

/**
 * Fetches the visitor's own accounts and account types, then their live
 * balances (one request per account, keyed off the first fetch) — shared by
 * every page that shows an account list with balances (`/accounts`, `/home`).
 */
export function useAccountsWithBalances(): {
  accounts: Account[] | null;
  types: AccountType[];
  balances: Record<string, number>;
} {
  const [accounts, setAccounts] = useState<Account[] | null>(null);
  const [types, setTypes] = useState<AccountType[]>([]);
  const [balances, setBalances] = useState<Record<string, number>>({});

  useEffect(() => {
    let cancelled = false;
    Promise.all([api.GET("/api/accounts"), api.GET("/api/account-types")]).then(
      ([accountsRes, typesRes]) => {
        if (cancelled) return;
        setAccounts(accountsRes.data ?? []);
        setTypes(typesRes.data ?? []);
      },
    );
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!accounts) return;
    let cancelled = false;
    Promise.all(
      accounts.map((account) =>
        api
          .GET("/api/accounts/{id}/balance", {
            params: { path: { id: account.id } },
          })
          .then(({ data }) => [account.id, data?.balance ?? 0] as const),
      ),
    ).then((results) => {
      if (cancelled) return;
      setBalances(Object.fromEntries(results));
    });
    return () => {
      cancelled = true;
    };
  }, [accounts]);

  return { accounts, types, balances };
}
