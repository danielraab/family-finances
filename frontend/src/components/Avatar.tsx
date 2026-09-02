import { colorFor, initials } from "../lib/monogram";

type AvatarProps = {
  id: string;
  email: string;
  displayName?: string | undefined;
  /** Rendered diameter in pixels. */
  size?: number | undefined;
};

/**
 * An initials monogram in a coloured circle — the fallback avatar for a user
 * with no picture. Decorative: the surrounding control carries the accessible
 * name.
 */
export function Avatar({ id, email, displayName, size = 32 }: AvatarProps) {
  return (
    <span
      aria-hidden="true"
      className={`inline-flex shrink-0 select-none items-center justify-center rounded-full font-semibold text-white ${colorFor(id)}`}
      style={{ width: size, height: size, fontSize: Math.round(size * 0.4) }}
    >
      {initials(displayName, email)}
    </span>
  );
}
