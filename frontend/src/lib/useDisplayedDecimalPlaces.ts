import { useEffect, useState } from "react";
import { api } from "../api/client";

/** Mirrors the backend's hardcoded default (see internal/settings). */
const DEFAULT_DISPLAYED_DECIMAL_PLACES = 2;

/**
 * The authenticated visitor's display-rounding preference
 * (`GET /api/settings`'s `displayed_decimal_places`, resolved server-side —
 * defaults to 2 when unset). Fetched independently by each page that needs
 * it, matching the rest of the frontend's no-shared-cache convention
 * (plain fetch in an effect); a tiny duplicate request per page is an
 * accepted cost of not introducing a client-side cache.
 */
export function useDisplayedDecimalPlaces(): number {
  const [places, setPlaces] = useState(DEFAULT_DISPLAYED_DECIMAL_PLACES);

  useEffect(() => {
    let cancelled = false;
    api.GET("/api/settings").then(({ data }) => {
      if (!cancelled && data) {
        setPlaces(data.displayed_decimal_places);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return places;
}
