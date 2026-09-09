import { entityColorVar } from "../lib/entityColors";
import { entityIconComponent } from "../lib/entityIcons";

type EntityIconProps = {
  /** Stored icon token; unknown or empty renders no glyph. */
  icon?: string | null | undefined;
  /** Stored palette colour token; unknown or empty renders a neutral chip. */
  color?: string | null | undefined;
  /** Badge side length in px (the glyph scales with it). Default 20. */
  size?: number | undefined;
  className?: string | undefined;
};

/**
 * The small icon/colour badge shown before an account or category name. A
 * coloured square (palette token → themed CSS var) holding the curated
 * glyph; a neutral square when only an icon is set; a bare colour chip when
 * only a colour is set. Renders nothing when neither is set, and degrades
 * to "no glyph" / "no colour" for tokens this build doesn't recognise — the
 * name always renders regardless (see `AccountLabel` / `CategoryLabel`).
 */
export function EntityIcon({
  icon,
  color,
  size = 20,
  className,
}: EntityIconProps) {
  const Glyph = entityIconComponent(icon ?? undefined);
  const colorValue = entityColorVar(color ?? undefined);

  if (!Glyph && !colorValue) {
    return null;
  }

  const base =
    "inline-flex shrink-0 items-center justify-center rounded-[5px] align-middle";

  if (colorValue) {
    return (
      <span
        className={`${base} text-white ${className ?? ""}`}
        style={{ width: size, height: size, backgroundColor: colorValue }}
      >
        {Glyph ? (
          <Glyph size={Math.round(size * 0.68)} strokeWidth={2.25} />
        ) : null}
      </span>
    );
  }

  return (
    <span
      className={`${base} bg-black/5 text-current dark:bg-white/10 ${className ?? ""}`}
      style={{ width: size, height: size }}
    >
      {Glyph ? (
        <Glyph size={Math.round(size * 0.68)} strokeWidth={2.25} />
      ) : null}
    </span>
  );
}
