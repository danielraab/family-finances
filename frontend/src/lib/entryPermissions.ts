import type { components } from "../api/schema";

type Account = components["schemas"]["Account"];
type Entry = components["schemas"]["Entry"];

type PermissionCarrier = Pick<Account, "permission">;

/**
 * entry_admin/owner may edit or delete any entry on the account; append
 * may edit or delete only what they themselves created; view (or append
 * on someone else's entry) is read-only — see account-entries' design.md.
 */
function fullTierAllows(
  acc: PermissionCarrier | null | undefined,
  createdBy: string,
  userId: string | undefined,
): boolean {
  return (
    acc !== null &&
    acc !== undefined &&
    (acc.permission === "entry_admin" ||
      acc.permission === "owner" ||
      (acc.permission === "append" && createdBy === userId))
  );
}

function atLeastAppend(acc: PermissionCarrier | null | undefined): boolean {
  return (
    acc !== null &&
    acc !== undefined &&
    (acc.permission === "append" ||
      acc.permission === "entry_admin" ||
      acc.permission === "owner")
  );
}

/**
 * Whether `userId` may edit `entry`, given their permission on its own
 * account and — for a self-transfer — on its counterpart account.
 *
 * A self-transfer's edit rule is stricter than an ordinary entry's:
 * append+ on *both* accounts, with no created_by exemption — see
 * account-entries' design.md.
 *
 * Shared by the entry edit page and the entry summary modal, so the two
 * can never disagree about who gets an edit affordance.
 */
export function canEditEntry(
  entry: Pick<Entry, "kind" | "created_by">,
  account: PermissionCarrier | null | undefined,
  toAccount: PermissionCarrier | null | undefined,
  userId: string | undefined,
): boolean {
  return entry.kind === "self_transfer"
    ? atLeastAppend(account) && atLeastAppend(toAccount)
    : fullTierAllows(account, entry.created_by, userId);
}

/**
 * Whether `userId` may delete `entry`. A self-transfer's delete rule is
 * looser than its edit rule — either account's ordinary tier is enough —
 * see account-entries' design.md.
 */
export function canDeleteEntry(
  entry: Pick<Entry, "kind" | "created_by">,
  account: PermissionCarrier | null | undefined,
  toAccount: PermissionCarrier | null | undefined,
  userId: string | undefined,
): boolean {
  return entry.kind === "self_transfer"
    ? fullTierAllows(account, entry.created_by, userId) ||
        fullTierAllows(toAccount, entry.created_by, userId)
    : fullTierAllows(account, entry.created_by, userId);
}
