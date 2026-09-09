import { useId, useState } from "react";

/** One series (a fixed identity across every category — e.g. "Income"). */
export type BarChartSeries = {
  label: string;
  /** Tailwind fill utility classes, e.g. "fill-[#008300] dark:fill-[#008300]" — see frontend/AGENTS.md's chart color convention. */
  fillClassName: string;
};

/** One category's values, one per entry in `series`, same order. */
export type BarChartDatum = {
  category: string;
  values: number[];
};

const CHART_HEIGHT = 220;
const BAR_MAX_THICKNESS = 24;
const BAR_GAP = 2; // the dataviz skill's "surface gap" between touching bars
const GROUP_GAP = 12;
const AXIS_LABEL_HEIGHT = 20;
const TICK_COUNT = 4;
// Room for the y-axis tick labels (left) and so the topmost tick's text
// isn't vertically clipped by the SVG's own viewBox (top) — a plain <svg>
// clips anything outside it by default.
const MARGIN_LEFT = 60;
const MARGIN_TOP = 10;

/** Rounds max up to a "nice" number (1/2/5 × a power of ten) for clean axis ticks. */
function niceMax(value: number): number {
  if (value <= 0) return 1;
  const exponent = Math.floor(Math.log10(value));
  const magnitude = 10 ** exponent;
  const fraction = value / magnitude;
  const niceFraction =
    fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 5 ? 5 : 10;
  return niceFraction * magnitude;
}

/**
 * A hand-rolled, dependency-free grouped bar chart — no charting library, per
 * frontend/AGENTS.md's chart convention. Presentational only: it takes data
 * and series definitions as props and renders nothing it fetched itself, so
 * it's reusable for any future N-category × M-series bar chart, not just
 * this one's income/outcome-per-month shape.
 *
 * Follows the dataviz skill's mark specs: bars capped at 24px thick with a
 * 4px rounded data-end (square at the baseline), a 2px surface gap between
 * touching bars, hairline recessive gridlines, a legend (always shown for
 * 2+ series), and a per-group hover/focus tooltip listing every series'
 * value (not just the one under the pointer).
 */
export function BarChart({
  series,
  data,
  formatValue,
}: {
  series: BarChartSeries[];
  data: BarChartDatum[];
  formatValue: (value: number) => string;
}) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const titleId = useId();

  const max = niceMax(
    Math.max(
      1,
      ...data.flatMap((d) => d.values.filter((v) => Number.isFinite(v))),
    ),
  );
  const ticks = Array.from(
    { length: TICK_COUNT + 1 },
    (_, i) => (max / TICK_COUNT) * i,
  );

  const groupWidth =
    series.length * BAR_MAX_THICKNESS + (series.length - 1) * BAR_GAP;
  const chartWidth = data.length * groupWidth + (data.length + 1) * GROUP_GAP;
  const plotHeight = CHART_HEIGHT - AXIS_LABEL_HEIGHT - MARGIN_TOP;

  return (
    <div className="flex flex-col gap-3">
      <ul className="flex flex-wrap gap-4 text-xs text-zinc-600 dark:text-zinc-400">
        {series.map((s) => (
          <li key={s.label} className="flex items-center gap-1.5">
            <span
              aria-hidden="true"
              className={`inline-block h-2.5 w-2.5 rounded-sm ${s.fillClassName}`}
            />
            {s.label}
          </li>
        ))}
      </ul>

      <div className="overflow-x-auto">
        <svg
          role="img"
          aria-labelledby={titleId}
          viewBox={`0 0 ${chartWidth + MARGIN_LEFT} ${CHART_HEIGHT}`}
          width={chartWidth + MARGIN_LEFT}
          height={CHART_HEIGHT}
          className="min-w-full"
        >
          <title id={titleId}>
            {series.map((s) => s.label).join(" / ")} by{" "}
            {data.map((d) => d.category).join(", ")}
          </title>

          {ticks.map((tick) => {
            const y = MARGIN_TOP + plotHeight - (tick / max) * plotHeight;
            return (
              <g key={tick}>
                <line
                  x1={MARGIN_LEFT}
                  x2={chartWidth + MARGIN_LEFT}
                  y1={y}
                  y2={y}
                  className="stroke-zinc-200 dark:stroke-zinc-800"
                  strokeWidth={1}
                />
                <text
                  x={MARGIN_LEFT - 8}
                  y={y}
                  textAnchor="end"
                  dominantBaseline="middle"
                  className="fill-zinc-500 text-[9px] dark:fill-zinc-400"
                >
                  {formatValue(tick)}
                </text>
              </g>
            );
          })}

          {data.map((d, groupIndex) => {
            const groupX =
              MARGIN_LEFT + GROUP_GAP + groupIndex * (groupWidth + GROUP_GAP);
            return (
              // biome-ignore lint/a11y/useSemanticElements: an SVG group has no native interactive equivalent; role+tabIndex is the correct fallback.
              <g
                key={d.category}
                tabIndex={0}
                role="button"
                aria-label={`${d.category}: ${series
                  .map((s, i) => `${s.label} ${formatValue(d.values[i] ?? 0)}`)
                  .join(", ")}`}
                onMouseEnter={() => setActiveIndex(groupIndex)}
                onMouseLeave={() => setActiveIndex(null)}
                onFocus={() => setActiveIndex(groupIndex)}
                onBlur={() => setActiveIndex(null)}
                className="cursor-default outline-none"
              >
                <rect
                  x={groupX}
                  y={MARGIN_TOP}
                  width={groupWidth}
                  height={plotHeight}
                  fill="transparent"
                />
                {series.map((s, seriesIndex) => {
                  const value = Math.max(0, d.values[seriesIndex] ?? 0);
                  const barHeight = max > 0 ? (value / max) * plotHeight : 0;
                  const barX =
                    groupX + seriesIndex * (BAR_MAX_THICKNESS + BAR_GAP);
                  const barY = MARGIN_TOP + plotHeight - barHeight;
                  return (
                    <rect
                      key={s.label}
                      x={barX}
                      y={barY}
                      width={BAR_MAX_THICKNESS}
                      height={Math.max(barHeight, 0)}
                      rx={4}
                      className={`${s.fillClassName} transition-opacity ${
                        activeIndex === null || activeIndex === groupIndex
                          ? "opacity-100"
                          : "opacity-40"
                      }`}
                    />
                  );
                })}
                <text
                  x={groupX + groupWidth / 2}
                  y={MARGIN_TOP + plotHeight + 14}
                  textAnchor="middle"
                  className="fill-zinc-500 text-[9px] dark:fill-zinc-400"
                >
                  {d.category}
                </text>
              </g>
            );
          })}
        </svg>
      </div>

      {activeIndex !== null && data[activeIndex] && (
        <div className="flex flex-col gap-1 rounded-md border border-black/10 bg-white px-3 py-2 text-xs shadow-sm dark:border-white/10 dark:bg-neutral-900">
          <span className="font-medium">{data[activeIndex].category}</span>
          {series.map((s, i) => (
            <span key={s.label} className="flex items-center gap-1.5">
              <span
                aria-hidden="true"
                className={`inline-block h-2 w-2 rounded-sm ${s.fillClassName}`}
              />
              <span className="text-zinc-500 dark:text-zinc-400">
                {s.label}
              </span>
              <span className="font-mono tabular-nums">
                {formatValue(data[activeIndex]?.values[i] ?? 0)}
              </span>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
