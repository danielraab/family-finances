import { useId, useLayoutEffect, useRef, useState } from "react";
import { niceMax, useContainerWidth, usePinnableSelection } from "./internal";

/** One series (a fixed identity across every category — e.g. "Income"). */
export type BarChartSeries = {
  label: string;
  /** Tailwind fill utility classes, e.g. "fill-[#008300] dark:fill-[#008300]" — see frontend/AGENTS.md's chart color convention. */
  fillClassName: string;
  /**
   * Optional: a muted/lighter fill for a second segment stacked on top of
   * this series' real segment, representing a projected (not-yet-real)
   * amount — see BarChartDatum.projectedValues. Only meaningful together
   * with projectedLabel and at least one non-zero projectedValues entry;
   * a series/datum with no projected data renders exactly as before this
   * capability existed.
   */
  projectedFillClassName?: string;
  /** The legend/tooltip label for this series' projected segment (e.g.
   * "Income (projected)"). Required alongside projectedFillClassName. */
  projectedLabel?: string;
};

/** One category's values, one per entry in `series`, same order. */
export type BarChartDatum = {
  category: string;
  values: number[];
  /** Parallel to `values` — an optional projected amount per series,
   * stacked as a second segment on top of the real one. Omitted, or all
   * zero/undefined, renders exactly as before this capability existed. */
  projectedValues?: number[];
};

const CHART_HEIGHT = 220;
const BAR_MAX_THICKNESS = 24;
// Below this the bars stop shrinking to fit and the chart keeps its intrinsic
// width instead, letting its `overflow-x-auto` wrapper scroll — the mobile /
// many-categories case.
const BAR_MIN_THICKNESS = 10;
const BAR_GAP = 2; // the dataviz skill's "surface gap" between touching bars
const GROUP_GAP = 12;
const AXIS_LABEL_HEIGHT = 20;
const TICK_COUNT = 4;
// Room for the y-axis tick labels (left) and so the topmost tick's text
// isn't vertically clipped by the SVG's own viewBox (top) — a plain <svg>
// clips anything outside it by default.
const MARGIN_LEFT = 60;
const MARGIN_TOP = 10;
const TOOLTIP_GAP = 8; // px between the tooltip and the top of the plot area

/**
 * A hand-rolled, dependency-free grouped bar chart — no charting library, per
 * frontend/AGENTS.md's chart convention. Presentational only: it takes data
 * and series definitions as props and renders nothing it fetched itself, so
 * it's reusable for any future N-category × M-series bar chart, not just
 * this one's income/outcome-per-month shape.
 *
 * Width is responsive: the chart measures its container and lays the groups
 * out to fill it, capping bar thickness at 24px. Only when that would force
 * bars below 10px (a narrow viewport, or lots of categories) does it fall
 * back to an intrinsic min-width and let the `overflow-x-auto` wrapper
 * scroll — so a desktop panel shows the whole chart, a phone scrolls it.
 *
 * Follows the dataviz skill's mark specs: a 4px rounded data-end (square at
 * the baseline), a 2px surface gap between touching bars, hairline recessive
 * gridlines, a legend (always shown for 2+ series), and a per-group
 * hover/focus tooltip listing every series' value (not just the one under
 * the pointer) — floated directly above the active group instead of a
 * permanent on-page value list. The tooltip is positioned from the active
 * group's actual rendered bounding box (not its SVG coordinates) so it stays
 * correctly anchored whether or not the `overflow-x-auto` wrapper is
 * scrolled. Clicking a group (tapping, on touch/mobile) pins that tooltip
 * and its highlight so they stay after the pointer leaves; clicking it
 * again, clicking another group, clicking away, or pressing Escape releases
 * the pin.
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
  // The group whose tooltip is transiently shown while hovered/focused, and
  // the one pinned by a click. A pin outlives the pointer; hover still wins
  // for the preview while it lasts.
  const { pinnedIndex, activeIndex, setHoverIndex, togglePin } =
    usePinnableSelection();
  const { containerRef, containerWidth } = useContainerWidth();
  const titleId = useId();

  // A wrapper around (not inside) the horizontally-scrolling container, so
  // the tooltip's containing block is never itself a scroll container —
  // nesting it inside `containerRef`'s `overflow-x-auto` element would clip
  // it whenever it pokes above the chart, since setting overflow-x alone
  // forces overflow-y to compute as "auto" too (a CSS quirk), turning that
  // element into a vertical clip box as well.
  const wrapperRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const [tooltipPosition, setTooltipPosition] = useState<{
    left: number;
    top: number;
  } | null>(null);

  const max = niceMax(
    Math.max(
      1,
      ...data.flatMap((d) =>
        d.values
          .map((v, i) => v + (d.projectedValues?.[i] ?? 0))
          .filter((v) => Number.isFinite(v)),
      ),
    ),
  );
  const ticks = Array.from(
    { length: TICK_COUNT + 1 },
    (_, i) => (max / TICK_COUNT) * i,
  );

  const groupCount = Math.max(data.length, 1);
  // Intrinsic floor: the width at which bars would hit BAR_MIN_THICKNESS. When
  // the container is at least this wide the chart fills it; when it's narrower
  // the chart stays this wide and its wrapper scrolls.
  const minGroupWidth =
    series.length * BAR_MIN_THICKNESS +
    Math.max(series.length - 1, 0) * BAR_GAP;
  const minChartWidth =
    MARGIN_LEFT + (groupCount + 1) * GROUP_GAP + groupCount * minGroupWidth;
  const chartWidth = Math.max(containerWidth, minChartWidth);

  const plotWidth = chartWidth - MARGIN_LEFT;
  const plotHeight = CHART_HEIGHT - AXIS_LABEL_HEIGHT - MARGIN_TOP;
  const groupWidth = (plotWidth - (groupCount + 1) * GROUP_GAP) / groupCount;
  const barThickness = Math.min(
    BAR_MAX_THICKNESS,
    (groupWidth - Math.max(series.length - 1, 0) * BAR_GAP) / series.length,
  );
  const barsWidth =
    series.length * barThickness + Math.max(series.length - 1, 0) * BAR_GAP;

  const tooltipVisible = activeIndex !== null && !!data[activeIndex];
  const tooltipDatum = activeIndex !== null ? data[activeIndex] : undefined;

  // Re-anchor the tooltip to the active group's actual rendered position
  // (and re-measure its own width, for horizontal clamping) whenever the
  // active group changes, and keep it glued there across scroll/resize
  // while it stays visible (a pinned tooltip can outlive the pointer, so
  // the page can still scroll under it). The tooltip node itself is always
  // rendered while `tooltipVisible` (just hidden until positioned, below)
  // so `tooltipRef.current` already exists the first time this runs —
  // otherwise, measuring its width would need a second effect pass, and
  // that pass would be skipped because [tooltipVisible, activeIndex]
  // wouldn't have changed since the first one.
  useLayoutEffect(() => {
    const wrapper = wrapperRef.current;
    const tooltipEl = tooltipRef.current;
    if (!tooltipVisible || activeIndex === null || !wrapper || !tooltipEl) {
      setTooltipPosition(null);
      return;
    }
    const groupEl = wrapper.querySelector<SVGGElement>(
      `[data-group-index="${activeIndex}"]`,
    );
    if (!groupEl) {
      setTooltipPosition(null);
      return;
    }
    const update = () => {
      const wrapperRect = wrapper.getBoundingClientRect();
      const groupRect = groupEl.getBoundingClientRect();
      const tooltipWidth = tooltipEl.offsetWidth;
      const anchorLeft =
        groupRect.left + groupRect.width / 2 - wrapperRect.left;
      // Clamp horizontally within the wrapper's own (visible) width so the
      // tooltip never spills off-screen for an edge group or a scrolled chart.
      const left = Math.min(
        Math.max(anchorLeft, tooltipWidth / 2),
        Math.max(wrapperRect.width - tooltipWidth / 2, tooltipWidth / 2),
      );
      setTooltipPosition({ left, top: groupRect.top - wrapperRect.top });
    };
    update();
    window.addEventListener("scroll", update, true);
    window.addEventListener("resize", update);
    return () => {
      window.removeEventListener("scroll", update, true);
      window.removeEventListener("resize", update);
    };
  }, [tooltipVisible, activeIndex]);

  return (
    <div className="flex flex-col gap-3">
      <ul className="flex flex-wrap gap-4 text-xs text-zinc-600 dark:text-zinc-400">
        {series.map((s) => (
          <li key={s.label} className="flex items-center gap-1.5">
            <svg
              aria-hidden="true"
              viewBox="0 0 10 10"
              className="inline-block h-2.5 w-2.5"
            >
              <rect width="10" height="10" rx="2" className={s.fillClassName} />
            </svg>
            {s.label}
          </li>
        ))}
        {series
          .filter((s) => s.projectedFillClassName && s.projectedLabel)
          .map((s) => (
            <li
              key={`${s.label}-projected`}
              className="flex items-center gap-1.5"
            >
              <svg
                aria-hidden="true"
                viewBox="0 0 10 10"
                className="inline-block h-2.5 w-2.5"
              >
                <rect
                  width="10"
                  height="10"
                  rx="2"
                  className={s.projectedFillClassName}
                />
              </svg>
              {s.projectedLabel}
            </li>
          ))}
      </ul>

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

            {ticks.map((tick) => {
              const y = MARGIN_TOP + plotHeight - (tick / max) * plotHeight;
              return (
                <g key={tick}>
                  <line
                    x1={MARGIN_LEFT}
                    x2={chartWidth}
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
              const barsStart = groupX + (groupWidth - barsWidth) / 2;
              return (
                // biome-ignore lint/a11y/useSemanticElements: an SVG group has no native interactive equivalent; role+tabIndex is the correct fallback.
                <g
                  key={d.category}
                  data-group-index={groupIndex}
                  tabIndex={0}
                  role="button"
                  aria-pressed={pinnedIndex === groupIndex}
                  aria-label={`${d.category}: ${series
                    .map((s, i) => {
                      const base = `${s.label} ${formatValue(d.values[i] ?? 0)}`;
                      const projected = d.projectedValues?.[i];
                      return s.projectedLabel && projected
                        ? `${base}, ${s.projectedLabel} ${formatValue(projected)}`
                        : base;
                    })
                    .join(", ")}`}
                  onMouseEnter={() => setHoverIndex(groupIndex)}
                  onMouseLeave={() => setHoverIndex(null)}
                  onFocus={() => setHoverIndex(groupIndex)}
                  onBlur={() => setHoverIndex(null)}
                  onClick={(e) => {
                    e.stopPropagation();
                    togglePin(groupIndex);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      e.stopPropagation();
                      togglePin(groupIndex);
                    }
                  }}
                  className="cursor-pointer outline-none"
                >
                  <rect
                    x={groupX}
                    y={MARGIN_TOP}
                    width={groupWidth}
                    height={plotHeight}
                    rx={4}
                    fill="transparent"
                    strokeWidth={1}
                    className={
                      pinnedIndex === groupIndex
                        ? "stroke-zinc-300 dark:stroke-zinc-600"
                        : "stroke-transparent"
                    }
                  />
                  {series.map((s, seriesIndex) => {
                    const value = Math.max(0, d.values[seriesIndex] ?? 0);
                    const projectedValue = Math.max(
                      0,
                      d.projectedValues?.[seriesIndex] ?? 0,
                    );
                    const barHeight = max > 0 ? (value / max) * plotHeight : 0;
                    const projectedHeight =
                      max > 0 ? (projectedValue / max) * plotHeight : 0;
                    const barX =
                      barsStart + seriesIndex * (barThickness + BAR_GAP);
                    const barY = MARGIN_TOP + plotHeight - barHeight;
                    const opacityClass =
                      activeIndex === null || activeIndex === groupIndex
                        ? "opacity-100"
                        : "opacity-40";
                    return (
                      <g key={s.label}>
                        <rect
                          x={barX}
                          y={barY}
                          width={barThickness}
                          height={Math.max(barHeight, 0)}
                          rx={4}
                          className={`${s.fillClassName} transition-opacity ${opacityClass}`}
                        />
                        {s.projectedFillClassName && projectedValue > 0 && (
                          <rect
                            x={barX}
                            y={barY - projectedHeight}
                            width={barThickness}
                            height={Math.max(projectedHeight, 0)}
                            rx={4}
                            className={`${s.projectedFillClassName} transition-opacity ${opacityClass}`}
                          />
                        )}
                      </g>
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
                  viewBox="0 0 8 8"
                  className="inline-block h-2 w-2"
                >
                  <rect
                    width="8"
                    height="8"
                    rx="2"
                    className={s.fillClassName}
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
              if (!s.projectedFillClassName || !s.projectedLabel) return null;
              return (
                <span
                  key={`${s.label}-projected`}
                  className="flex items-center gap-1.5"
                >
                  <svg
                    aria-hidden="true"
                    viewBox="0 0 8 8"
                    className="inline-block h-2 w-2"
                  >
                    <rect
                      width="8"
                      height="8"
                      rx="2"
                      className={s.projectedFillClassName}
                    />
                  </svg>
                  <span className="text-zinc-500 dark:text-zinc-400">
                    {s.projectedLabel}
                  </span>
                  <span className="font-mono tabular-nums">
                    {formatValue(tooltipDatum.projectedValues?.[i] ?? 0)}
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
