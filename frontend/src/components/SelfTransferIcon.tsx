/** The two-way arrows glyph used app-wide as the "self-transfer" affordance
 * — see SelfTransferBadge.tsx. Distinct from RecurringIcon's looping-repeat
 * glyph, since a transfer moves between two accounts once, it doesn't
 * recur. */
export function SelfTransferIcon({
  width = 14,
  height = 14,
}: {
  width?: number;
  height?: number;
}) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={width}
      height={height}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="m17 3 4 4-4 4" />
      <path d="M3 7h18" />
      <path d="m7 21-4-4 4-4" />
      <path d="M21 17H3" />
    </svg>
  );
}
