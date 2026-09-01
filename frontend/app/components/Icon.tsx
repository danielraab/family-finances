type IconProps = {
  /** Rendered width/height in pixels. */
  size?: number;
  className?: string;
  title?: string;
};

/**
 * The Family Finances mark: a house with a Euro sign cut out, on a deep-green
 * tile. Kept in sync with `app/icon.svg` (the favicon source).
 */
export function Icon({
  size = 32,
  className,
  title = "Family Finances",
}: IconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 64 64"
      width={size}
      height={size}
      role="img"
      aria-label={title}
      className={className}
    >
      <title>{title}</title>
      <defs>
        <mask id="ff-euro-cut">
          <path d="M32 8 L56 30 L56 56 L8 56 L8 30 Z" fill="#fff" />
          <g fill="none" stroke="#000" strokeWidth="5" strokeLinecap="round">
            <path d="M43 27 A 13 13 0 1 0 43 45" />
            <path d="M17 33 H38" />
            <path d="M17 40 H38" />
          </g>
        </mask>
      </defs>
      <rect width="64" height="64" rx="14" fill="#15803d" />
      <path
        d="M32 8 L56 30 L56 56 L8 56 L8 30 Z"
        fill="#fff"
        mask="url(#ff-euro-cut)"
      />
    </svg>
  );
}
