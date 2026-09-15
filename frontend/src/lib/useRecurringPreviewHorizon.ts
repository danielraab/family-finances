import { useEffect, useState } from "react";
import { api } from "../api/client";
import {
  DEFAULT_RECURRING_PREVIEW_HORIZON,
  type RecurringPreviewHorizon,
} from "./recurringPreview";

/**
 * The authenticated visitor's recurring-preview horizon preference
 * (`GET /api/settings`'s `recurring_preview_horizon`, resolved server-side —
 * defaults to "end_of_this_month" when unset). Fetched independently by
 * each page that needs it, matching `useWeekStart`'s no-shared-cache
 * convention.
 */
export function useRecurringPreviewHorizon(): RecurringPreviewHorizon {
  const [horizon, setHorizon] = useState<RecurringPreviewHorizon>(
    DEFAULT_RECURRING_PREVIEW_HORIZON,
  );

  useEffect(() => {
    let cancelled = false;
    api.GET("/api/settings").then(({ data }) => {
      if (!cancelled && data) {
        setHorizon(data.recurring_preview_horizon);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return horizon;
}
