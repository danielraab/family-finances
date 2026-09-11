import { useTranslation } from "react-i18next";
import type { components } from "../api/schema";

type Tag = components["schemas"]["Tag"];

type TagLabelProps = {
  tag: Pick<Tag, "name" | "shared" | "owner_name">;
  /** Classes for the wrapper (e.g. `font-medium`). */
  className?: string | undefined;
};

function SharedGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={14}
      height={14}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M16 3.5a3 3 0 1 0 0 6 3 3 0 0 0 0-6ZM8 9.5a3 3 0 1 0 0 6 3 3 0 0 0 0-6ZM16 14.5a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z" />
      <path d="m10.7 11 4.6-2.6M10.7 13l4.6 2.6" />
    </svg>
  );
}

/**
 * A tag's name, followed by a small shared-indicator icon whenever the tag
 * is shared (i.e. the viewer isn't its real owner) — never shown to the
 * real owner themselves, even when they've shared it with others. Unlike
 * CategoryLabel's persistent owner-name badge, the icon is the only
 * always-visible indicator here; the owner's name surfaces only in the
 * icon's hover/title tooltip, by deliberate design decision (see
 * design.md's TagLabel decision).
 */
export function TagLabel({ tag, className }: TagLabelProps) {
  const { t } = useTranslation();
  return (
    <span
      className={`inline-flex min-w-0 items-center gap-1.5 ${className ?? ""}`}
    >
      <span className="min-w-0 truncate">{tag.name}</span>
      {tag.shared && (
        <span
          className="inline-flex shrink-0 items-center text-zinc-500 dark:text-zinc-400"
          title={t("tags.shared.badgeTitle", {
            owner: tag.owner_name ?? "",
          })}
        >
          <SharedGlyph />
        </span>
      )}
    </span>
  );
}
