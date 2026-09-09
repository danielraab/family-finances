import type { components } from "../api/schema";
import { EntityIcon } from "./EntityIcon";

type Account = components["schemas"]["Account"];

type AccountLabelProps = {
  account: Pick<Account, "title" | "icon" | "color">;
  /** Badge size in px. Default 20. */
  iconSize?: number | undefined;
  /** Classes for the wrapper (e.g. `font-medium`). */
  className?: string | undefined;
};

/**
 * An account's name, preceded by its icon/colour badge when it has one.
 * Used everywhere an account is shown by name outside a native `<select>`.
 */
export function AccountLabel({
  account,
  iconSize,
  className,
}: AccountLabelProps) {
  return (
    <span className={`inline-flex items-center gap-1.5 ${className ?? ""}`}>
      <EntityIcon icon={account.icon} color={account.color} size={iconSize} />
      {account.title}
    </span>
  );
}
