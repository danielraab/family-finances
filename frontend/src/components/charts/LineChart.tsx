import { useId, useRef } from "react";
import {
  niceExtent,
  useContainerWidth,
  useOverlayTooltipPosition,
  usePinnableSelection,
} from "./internal";

/** One line (a fixed identity across every point — e.g. an account's balance). */
export type LineChartSeries = {
  label: string;
  /** Tailwind stroke utility classes for the line, e.g. "stroke-[#4269d0] dark:stroke-[#a5b8ef]" — see frontend/AGENTS.md's chart color convention. */
  strokeClassName: string;
  /** Tailwind fill utility classes for the point dots, matching the stroke color, e.g. "fill-[#4269d0] dark:fill-[#a5b8ef]". Passed as its own literal string so Tailwind's scanner emits the class. */
  dotClassName: string;
  /**
   * Optional: a muted/lighter stroke for a second, dashed line continuing
   * this series' real line, representing a projected (not-yet-real) value
   * — see LineChartDatum.projectedValues. Only meaningful together with
   * projectedLabel and at least one datum with a defined projectedValues
   * entry; a series/datum with no projected data renders exactly as before
   * this capability existed.
   */
  projectedStrokeClassName?: string;
  /** The legend/tooltip label for this series' projected line (e.g.
   * "Balance (projected)"). Required alongside projectedStrokeClassName. */
  projectedLabel?: string;
};

/**
 * One point's values, one per entry in `series`, same order. `category` is
 * the x-axis label and doubles as the point's React key, so a caller must
 * keep it unique across `data` (e.g. a month view whose closing boundary
 * lands on the 1st should label that point distinctly, not "1" again).
 */
export type LineChartDatum = {
  category: string;
  values: number[];
  /** Parallel to `values` — an optional projected value per series, drawn
   * as a second, dashed line. Omitted (undefined) at a given index draws
   * no projected segment there; a caller should make the projected value
   * equal the real value at the first index it defines one, so the dashed
   * line visually continues from the solid one rather than floating
   * separately (see lib/recurringPreview.ts's
   * cumulativePreviewDeltaByPeriod, one caller that builds this). */
  projectedValues?: number[];
};

const CHART_HEIGHT = 220;
const AXIS_LABEL_HEIGHT = 20;
// Room for the y-axis tick labels (left) and so the topmost tick's text isn't
// clipped by the SVG's own viewBox (top) — a plain <svg> clips anything
// outside it.
const MARGIN_LEFT = 60;
const MARGIN_TOP = 10;
const PLOT_INSET = 8; // keeps the first/last point off the plot edges
const MIN_STEP = 14; // min horizontal spacing per point before the wrapper scrolls
const DOT_RADIUS = 2.5;
const TOOLTIP_GAP = 8; // px between the tooltip and the top of the plot area

/**
 * A hand-rolled, dependency-free line chart — no charting library, per
 * frontend/AGENTS.md's chart convention. Presentational only: data and
 * series definitions come in as props, nothing is fetched inside, so it's
 * reusable for any N-point × M-series line, not just the account details
 * page's running-balance shape.
 *
 * The y-axis is signed and framed to the data (a balance around 50k fills
 * the plot rather than hugging the top); when the range straddles 0 a
 * stronger zero rule line is drawn. The line is a step — a balance holds
 * flat between the entries that move it, so a straight diagonal between two
 * samples would imply money moving on a day nothing happened.
 *
 * Width is responsive: the chart measures its container and lays the points
 * out to fill it. Only when that would push points below ~14px apart does
 * it keep an intrinsic min-width and let the `overflow-x-auto` wrapper
 * scroll. A per-point hover/focus tooltip listing every series' value at
 * that point floats directly above the active point — positioned from its
 * actual rendered bounding box via `useOverlayTooltipPosition`, mirroring
 * `BarChart`'s own tooltip exactly. Clicking a point (tapping, on
 * touch/mobile) pins that tooltip so it stays after the pointer leaves;
 * clicking it again, clicking another point, clicking away, or pressing
 * Escape releases the pin.
 */
export function LineChart({
  series,
  data,
  formatValue,
}: {
  series: LineChartSeries[];
  data: LineChartDatum[];
  formatValue: (value: number) => string;
}) {
  const { pinnedIndex, activeIndex, setHoverIndex, togglePin } =
    usePinnableSelection();
  const { containerRef, containerWidth } = useContainerWidth();
  const titleId = useId();

  // A wrapper around (not inside) the horizontally-scrolling container, so
  // the tooltip's containing block is never itself a scroll container —
  // see BarChart's identical comment for why.
  const wrapperRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  const finiteValues = data.flatMap((d) =>
    [...d.values, ...(d.projectedValues ?? [])].filter((v) =>
      Number.isFinite(v),
    ),
  );
  const domain = niceExtent(
    finiteValues.length ? Math.min(...finiteValues) : 0,
    finiteValues.length ? Math.max(...finiteValues) : 0,
  );
  const span = domain.max - domain.min || 1;

  const pointCount = Math.max(data.length, 1);
  const minChartWidth =
    MARGIN_LEFT + 2 * PLOT_INSET + (pointCount - 1) * MIN_STEP;
  const chartWidth = Math.max(containerWidth, minChartWidth);

  const plotWidth = chartWidth - MARGIN_LEFT;
  const plotHeight = CHART_HEIGHT - AXIS_LABEL_HEIGHT - MARGIN_TOP;
  const innerWidth = plotWidth - 2 * PLOT_INSET;

  const xAt = (i: number) =>
    MARGIN_LEFT +
    PLOT_INSET +
    (pointCount > 1 ? (innerWidth * i) / (pointCount - 1) : innerWidth / 2);
  const yAt = (value: number) =>
    MARGIN_TOP + plotHeight - ((value - domain.min) / span) * plotHeight;

  const tooltipVisible = activeIndex !== null && !!data[activeIndex];
  const tooltipDatum = activeIndex !== null ? data[activeIndex] : undefined;

  const tooltipPosition = useOverlayTooltipPosition({
    wrapperRef,
    tooltipRef,
    visible: tooltipVisible,
    anchorSelector:
      activeIndex !== null ? `[data-point-index="${activeIndex}"]` : null,
  });

  return (
    <div className="flex flex-col gap-3">
      {(series.length > 1 ||
        series.some((s) => s.projectedStrokeClassName && s.projectedLabel)) && (
        <ul className="flex flex-wrap gap-4 text-xs text-zinc-600 dark:text-zinc-400">
          {series.map((s) => (
            <li key={s.label} className="flex items-center gap-1.5">
              <svg
                aria-hidden="true"
                viewBox="0 0 12 12"
                className="inline-block h-2.5 w-2.5"
              >
                <line
                  x1="0"
                  y1="6"
                  x2="12"
                  y2="6"
                  strokeWidth="2"
                  className={s.strokeClassName}
                />
              </svg>
              {s.label}
            </li>
          ))}
          {series
            .filter((s) => s.projectedStrokeClassName && s.projectedLabel)
            .map((s) => (
              <li
                key={`${s.label}-projected`}
                className="flex items-center gap-1.5"
              >
                <svg
                  aria-hidden="true"
                  viewBox="0 0 12 12"
                  className="inline-block h-2.5 w-2.5"
                >
                  <line
                    x1="0"
                    y1="6"
                    x2="12"
                    y2="6"
                    strokeWidth="2"
                    strokeDasharray="3 2"
                    className={s.projectedStrokeClassName}
                  />
                </svg>
                {s.projectedLabel}
              </li>
            ))}
        </ul>
      )}

      <div ref={wrapperRef} className="relative">
        <div ref={containerRef} className="overflow-x-auto">
          <svg
            role="img"
            aria-labelledby={titleId}
            viewBox={`0 0 ${chartWidth} ${CHART_HEIGHT}`}
            width={chartWidth}
            height={CHART_HEIGHT}
            className="block"
          >
            <title id={titleId}>
              {series.map((s) => s.label).join(" / ")} by{" "}
              {data.map((d) => d.category).join(", ")}
            </title>

            {domain.ticks.map((tick) => {
              const y = yAt(tick);
              const isZero = tick === 0 && domain.zeroInDomain;
              return (
                <g key={tick}>
                  <line
                    x1={MARGIN_LEFT}
                    x2={chartWidth}
                    y1={y}
                    y2={y}
                    className={
                      isZero
                        ? "stroke-zinc-300 dark:stroke-zinc-700"
                        : "stroke-zinc-200 dark:stroke-zinc-800"
                    }
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

            {series.map((s, seriesIndex) => {
              // stepAfter: hold each point's value flat until the next x, then
              // jump — the truthful shape for a balance.
              const segments = data.map((d, i) => {
                const y = yAt(d.values[seriesIndex] ?? 0);
                const x = xAt(i);
                if (i === 0) return `M ${x} ${y}`;
                const prevY = yAt(data[i - 1]?.values[seriesIndex] ?? 0);
                return `L ${x} ${prevY} L ${x} ${y}`;
              });
              return (
                <path
                  key={s.label}
                  d={segments.join(" ")}
                  fill="none"
                  strokeWidth={2}
                  strokeLinejoin="round"
                  className={`${s.strokeClassName} transition-opacity ${
                    activeIndex === null ? "opacity-100" : "opacity-70"
                  }`}
                />
              );
            })}

            {series.map((s, seriesIndex) => {
              if (!s.projectedStrokeClassName) return null;
              // Only the indices this series actually projects a value for
              // (in practice a trailing contiguous run starting "today") —
              // no bridging logic needed here since the caller guarantees
              // the first such index's value equals the real line's value
              // there, so this path starts exactly on the solid one.
              const projectedIndexes = data
                .map((d, i) =>
                  d.projectedValues?.[seriesIndex] !== undefined ? i : null,
                )
                .filter((i): i is number => i !== null);
              if (projectedIndexes.length === 0) return null;
              const segments = projectedIndexes.map((i, k) => {
                const value = data[i]?.projectedValues?.[seriesIndex] ?? 0;
                const y = yAt(value);
                const x = xAt(i);
                if (k === 0) return `M ${x} ${y}`;
                const prevI = projectedIndexes[k - 1] ?? i;
                const prevValue =
                  data[prevI]?.projectedValues?.[seriesIndex] ?? 0;
                const prevY = yAt(prevValue);
                return `L ${x} ${prevY} L ${x} ${y}`;
              });
              return (
                <path
                  key={`${s.label}-projected`}
                  d={segments.join(" ")}
                  fill="none"
                  strokeWidth={2}
                  strokeLinejoin="round"
                  strokeDasharray="5 4"
                  className={`${s.projectedStrokeClassName} transition-opacity ${
                    activeIndex === null ? "opacity-100" : "opacity-70"
                  }`}
                />
              );
            })}

            {data.map((d, pointIndex) => {
              const x = xAt(pointIndex);
              const isActive = activeIndex === pointIndex;
              return (
                // biome-ignore lint/a11y/useSemanticElements: an SVG group has no native interactive equivalent; role+tabIndex is the correct fallback.
                <g
                  key={d.category}
                  data-point-index={pointIndex}
                  tabIndex={0}
                  role="button"
                  aria-pressed={pinnedIndex === pointIndex}
                  aria-label={`${d.category}: ${series
                    .map((s, i) => {
                      const base = `${s.label} ${formatValue(d.values[i] ?? 0)}`;
                      const projected = d.projectedValues?.[i];
                      return s.projectedLabel && projected !== undefined
                        ? `${base}, ${s.projectedLabel} ${formatValue(projected)}`
                        : base;
                    })
                    .join(", ")}`}
                  onMouseEnter={() => setHoverIndex(pointIndex)}
                  onMouseLeave={() => setHoverIndex(null)}
                  onFocus={() => setHoverIndex(pointIndex)}
                  onBlur={() => setHoverIndex(null)}
                  onClick={(e) => {
                    e.stopPropagation();
                    togglePin(pointIndex);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      e.stopPropagation();
                      togglePin(pointIndex);
                    }
                  }}
                  className="cursor-pointer outline-none"
                >
                  {/* full-height band so the whole column is a hover/click target */}
                  <rect
                    x={x - Math.max(innerWidth / pointCount / 2, MIN_STEP / 2)}
                    y={MARGIN_TOP}
                    width={Math.max(innerWidth / pointCount, MIN_STEP)}
                    height={plotHeight}
                    fill="transparent"
                  />
                  {isActive && (
                    <line
                      x1={x}
                      x2={x}
                      y1={MARGIN_TOP}
                      y2={MARGIN_TOP + plotHeight}
                      className="stroke-zinc-300 dark:stroke-zinc-600"
                      strokeWidth={1}
                    />
                  )}
                  {series.map((s, seriesIndex) => (
                    <circle
                      key={s.label}
                      cx={x}
                      cy={yAt(d.values[seriesIndex] ?? 0)}
                      r={isActive ? DOT_RADIUS + 1.5 : DOT_RADIUS}
                      className={s.dotClassName}
                    />
                  ))}
                  {pointIndex % Math.ceil(pointCount / 8) === 0 && (
                    <text
                      x={x}
                      y={MARGIN_TOP + plotHeight + 14}
                      textAnchor="middle"
                      className="fill-zinc-500 text-[9px] dark:fill-zinc-400"
                    >
                      {d.category}
                    </text>
                  )}
                </g>
              );
            })}
          </svg>
        </div>

        {tooltipVisible && tooltipDatum && (
          <div
            ref={tooltipRef}
            role="status"
            style={
              tooltipPosition
                ? {
                    left: tooltipPosition.left,
                    top: tooltipPosition.top - TOOLTIP_GAP,
                  }
                : undefined
            }
            className={`pointer-events-none absolute z-10 flex -translate-x-1/2 -translate-y-full flex-col gap-1 rounded-md border border-black/10 bg-white px-3 py-2 text-xs shadow-md dark:border-white/10 dark:bg-neutral-900${
              tooltipPosition ? "" : " invisible"
            }`}
          >
            <span className="font-medium">{tooltipDatum.category}</span>
            {series.map((s, i) => (
              <span key={s.label} className="flex items-center gap-1.5">
                <svg
                  aria-hidden="true"
                  viewBox="0 0 12 12"
                  className="inline-block h-2 w-2"
                >
                  <line
                    x1="0"
                    y1="6"
                    x2="12"
                    y2="6"
                    strokeWidth="2"
                    className={s.strokeClassName}
                  />
                </svg>
                <span className="text-zinc-500 dark:text-zinc-400">
                  {s.label}
                </span>
                <span className="font-mono tabular-nums">
                  {formatValue(tooltipDatum.values[i] ?? 0)}
                </span>
              </span>
            ))}
            {series.map((s, i) => {
              if (!s.projectedStrokeClassName || !s.projectedLabel) return null;
              const projected = tooltipDatum.projectedValues?.[i];
              if (projected === undefined) return null;
              return (
                <span
                  key={`${s.label}-projected`}
                  className="flex items-center gap-1.5"
                >
                  <svg
                    aria-hidden="true"
                    viewBox="0 0 12 12"
                    className="inline-block h-2 w-2"
                  >
                    <line
                      x1="0"
                      y1="6"
                      x2="12"
                      y2="6"
                      strokeWidth="2"
                      strokeDasharray="3 2"
                      className={s.projectedStrokeClassName}
                    />
                  </svg>
                  <span className="text-zinc-500 dark:text-zinc-400">
                    {s.projectedLabel}
                  </span>
                  <span className="font-mono tabular-nums">
                    {formatValue(projected)}
                  </span>
                </span>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
