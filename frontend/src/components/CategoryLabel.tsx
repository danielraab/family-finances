import type { components } from "../api/schema";
import { EntityIcon } from "./EntityIcon";

type Category = components["schemas"]["Category"];

type CategoryLabelProps = {
  category: Pick<Category, "name" | "icon" | "color">;
  /** Badge size in px. Default 20. */
  iconSize?: number | undefined;
  /** Classes for the wrapper. */
  className?: string | undefined;
};

/**
 * A category's name, preceded by its icon/colour badge when it has one.
 * Used everywhere a category is shown by name outside a native `<select>`.
 */
export function CategoryLabel({
  category,
  iconSize,
  className,
}: CategoryLabelProps) {
  return (
    <span className={`inline-flex items-center gap-1.5 ${className ?? ""}`}>
      <EntityIcon icon={category.icon} color={category.color} size={iconSize} />
      {category.name}
    </span>
  );
}
