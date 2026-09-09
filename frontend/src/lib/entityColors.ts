/**
 * The fixed palette an account or category `color` token may name. Tokens
 * are opaque on the wire (the backend never interprets them); this module
 * is the single client-side source of truth for what they mean.
 *
 * The eight hues are the project's validated categorical palette (see the
 * `dataviz` skill's `references/palette.md`), each with a light- and a
 * dark-theme value. The actual hex lives in `src/styles.css` as
 * `--entity-<token>` custom properties that switch on the `.dark` class, so
 * a colour adapts to the theme with no JS re-render; the map below mirrors
 * those values for any code (tests, future chart wiring) that needs the raw
 * hex, and MUST be kept in sync with the stylesheet.
 */
export const ENTITY_COLORS = {
  blue: { light: "#2a78d6", dark: "#3987e5" },
  orange: { light: "#eb6834", dark: "#d95926" },
  aqua: { light: "#1baf7a", dark: "#199e70" },
  yellow: { light: "#eda100", dark: "#c98500" },
  magenta: { light: "#e87ba4", dark: "#d55181" },
  green: { light: "#008300", dark: "#008300" },
  violet: { light: "#4a3aa7", dark: "#9085e9" },
  red: { light: "#e34948", dark: "#e66767" },
} as const;

export type EntityColorToken = keyof typeof ENTITY_COLORS;

/** Ordered list of palette tokens, for the picker's swatch row. */
export const ENTITY_COLOR_TOKENS = Object.keys(
  ENTITY_COLORS,
) as EntityColorToken[];

/** Narrows an arbitrary string to a known palette token. */
export function isEntityColorToken(value: string): value is EntityColorToken {
  return Object.hasOwn(ENTITY_COLORS, value);
}

/**
 * A CSS value for a stored `color` token — a reference to the themed custom
 * property in `styles.css` — or `undefined` when the token is empty or not
 * part of the current palette (an older/newer client wrote it). Callers
 * treat `undefined` as "no colour".
 */
export function entityColorVar(token: string | undefined): string | undefined {
  if (!token || !isEntityColorToken(token)) {
    return undefined;
  }
  return `var(--entity-${token})`;
}
