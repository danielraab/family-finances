import {
  type RefObject,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

/**
 * Shared scaffolding for the hand-rolled charts in this directory
 * (`BarChart`, `LineChart`) — deliberately not a charting library, per
 * frontend/AGENTS.md. Pure layout math plus a few small hooks; every chart
 * stays presentational and owns its own SVG.
 */

const TICK_COUNT = 4;

/** Rounds `value` up to a "nice" number (1/2/5 × a power of ten). */
export function niceMax(value: number): number {
  if (value <= 0) return 1;
  const exponent = Math.floor(Math.log10(value));
  const magnitude = 10 ** exponent;
  const fraction = value / magnitude;
  const niceFraction =
    fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 5 ? 5 : 10;
  return niceFraction * magnitude;
}

/**
 * A signed axis domain framed to the data — both bounds rounded outward to
 * a multiple of one "nice" step, with the tick values in between. Unlike a
 * bar chart (whose baseline is always 0), a line is framed to its own
 * range: a balance hovering around 50k fills the plot instead of hugging
 * the top. When the rounded range straddles 0, 0 falls on a tick and
 * `zeroInDomain` is true so the caller can draw a rule line there.
 */
export function niceExtent(
  min: number,
  max: number,
): { min: number; max: number; ticks: number[]; zeroInDomain: boolean } {
  if (min === max) {
    const pad = Math.max(Math.abs(min) * 0.1, 1);
    min -= pad;
    max += pad;
  }
  const step = niceMax((max - min) / TICK_COUNT);
  const lo = Math.floor(min / step) * step;
  const hi = Math.ceil(max / step) * step;
  const ticks: number[] = [];
  for (let k = Math.round(lo / step); k <= Math.round(hi / step); k++) {
    ticks.push(k * step);
  }
  return { min: lo, max: hi, ticks, zeroInDomain: lo < 0 && hi > 0 };
}

/**
 * Measures a container element's width via `ResizeObserver` so a chart can
 * lay itself out to fill the space it's given. Returns the ref to attach
 * and the last observed width (0 until the first observation).
 */
export function useContainerWidth() {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) setContainerWidth(entry.contentRect.width);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return { containerRef, containerWidth };
}

/**
 * The hover-preview / click-to-pin selection state shared by the charts. A
 * pointer hover (or keyboard focus) previews an index; a click pins it so
 * the tooltip and highlight outlive the pointer. While something is
 * pinned, a click anywhere else — the pinning clicks must
 * `stopPropagation` before they bubble to `window` — or the Escape key
 * releases it. `activeIndex` is the hover preview when there is one, else
 * the pinned index.
 */
export function usePinnableSelection() {
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [pinnedIndex, setPinnedIndex] = useState<number | null>(null);
  const activeIndex = hoverIndex ?? pinnedIndex;

  useEffect(() => {
    if (pinnedIndex === null) return;
    const release = () => setPinnedIndex(null);
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setPinnedIndex(null);
    };
    window.addEventListener("click", release);
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("click", release);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [pinnedIndex]);

  const togglePin = (index: number) =>
    setPinnedIndex((prev) => (prev === index ? null : index));

  return {
    hoverIndex,
    pinnedIndex,
    activeIndex,
    setHoverIndex,
    togglePin,
  };
}

/**
 * Positions a chart's floating overlay tooltip against the element
 * matching `anchorSelector` inside `wrapperRef` — the container-relative,
 * scroll/resize-tracking, horizontally-clamped math `BarChart` originated
 * for its own tooltip, generalized so `LineChart` can anchor to a point
 * instead of a bar group without duplicating it. `wrapperRef` must be a
 * non-scrolling `position: relative` ancestor of both the anchor element
 * and `tooltipRef` (never the `overflow-x-auto` container itself — that
 * would clip a tooltip poking above the chart, since setting `overflow-x`
 * alone forces `overflow-y` to compute as `auto` too).
 *
 * `anchorSelector` is `null` when nothing is active; the hook then returns
 * `null` and does nothing else. Otherwise it measures the anchor and
 * tooltip via `getBoundingClientRect()`, clamps `left` within the
 * wrapper's own visible width, anchors `top` to the element's own top
 * edge, and re-measures on `scroll`/`resize` while `visible` stays true
 * (a pinned tooltip can outlive the pointer, so the page can still scroll
 * under it).
 */
export function useOverlayTooltipPosition({
  wrapperRef,
  tooltipRef,
  visible,
  anchorSelector,
}: {
  wrapperRef: RefObject<HTMLDivElement | null>;
  tooltipRef: RefObject<HTMLDivElement | null>;
  visible: boolean;
  anchorSelector: string | null;
}): { left: number; top: number } | null {
  const [position, setPosition] = useState<{
    left: number;
    top: number;
  } | null>(null);

  useLayoutEffect(() => {
    const wrapper = wrapperRef.current;
    const tooltipEl = tooltipRef.current;
    if (!visible || anchorSelector === null || !wrapper || !tooltipEl) {
      setPosition(null);
      return;
    }
    const anchorEl = wrapper.querySelector<Element>(anchorSelector);
    if (!anchorEl) {
      setPosition(null);
      return;
    }
    const update = () => {
      const wrapperRect = wrapper.getBoundingClientRect();
      const anchorRect = anchorEl.getBoundingClientRect();
      const tooltipWidth = tooltipEl.offsetWidth;
      const anchorLeft =
        anchorRect.left + anchorRect.width / 2 - wrapperRect.left;
      // Clamp horizontally within the wrapper's own (visible) width so the
      // tooltip never spills off-screen for an edge anchor or a scrolled chart.
      const left = Math.min(
        Math.max(anchorLeft, tooltipWidth / 2),
        Math.max(wrapperRect.width - tooltipWidth / 2, tooltipWidth / 2),
      );
      setPosition({ left, top: anchorRect.top - wrapperRect.top });
    };
    update();
    window.addEventListener("scroll", update, true);
    window.addEventListener("resize", update);
    return () => {
      window.removeEventListener("scroll", update, true);
      window.removeEventListener("resize", update);
    };
  }, [wrapperRef, tooltipRef, visible, anchorSelector]);

  return position;
}
