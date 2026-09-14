import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { WeekStart } from "./dateRangePresets";

/** Mirrors the backend's hardcoded default (see internal/settings). */
const DEFAULT_WEEK_START: WeekStart = "monday";

/**
 * The authenticated visitor's week-start preference
 * (`GET /api/settings`'s `week_start`, resolved server-side — defaults to
 * "monday" when unset). Fetched independently by each page that needs it,
 * matching `useDisplayedDecimalPlaces`'s no-shared-cache convention.
 */
export function useWeekStart(): WeekStart {
  const [weekStart, setWeekStart] = useState<WeekStart>(DEFAULT_WEEK_START);

  useEffect(() => {
    let cancelled = false;
    api.GET("/api/settings").then(({ data }) => {
      if (!cancelled && data) {
        setWeekStart(data.week_start);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return weekStart;
}
