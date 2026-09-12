import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import L from "leaflet";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { type Coordinates, formatCoordinates } from "../lib/location";
import { OSM_ATTRIBUTION, OSM_TILE_URL } from "./leafletSetup";

type LocationPickerModalProps = {
  open: boolean;
  onClose: () => void;
  onConfirm: (value: string) => void;
  initial: Coordinates | null;
};

const WORLD_VIEW: Coordinates = { lat: 20, lng: 0 };
const WORLD_ZOOM = 2;
const PIN_ZOOM = 15;

// Interactive: click-to-place, draggable marker, confirmed explicitly —
// distinct from LocationPreviewModal's read-only view (see design.md's
// "two separate map surfaces" decision). Centers on the device's current
// position when no initial value is given and geolocation succeeds,
// otherwise falls back to (and stays at) the world view. No address
// search, by design.
export function LocationPickerModal({
  open,
  onClose,
  onConfirm,
  initial,
}: LocationPickerModalProps) {
  const { t } = useTranslation();
  // A plain useRef (paired with an effect keyed on `open`) is unreliable
  // here: Dialog mounts its panel through a portal, and the ref isn't
  // guaranteed to be attached by the time an effect keyed only on `open`
  // runs. A callback ref stored in state re-renders (and so re-runs the
  // effect) exactly when the DOM node itself becomes available, which is
  // the timing that actually matters.
  const [container, setContainer] = useState<HTMLDivElement | null>(null);
  const markerRef = useRef<L.Marker | null>(null);
  const [selected, setSelected] = useState<Coordinates | null>(initial);

  useEffect(() => {
    if (!open || !container) return;
    setSelected(initial);
    markerRef.current = null;
    let cancelled = false;

    const start = initial ?? WORLD_VIEW;
    const map = L.map(container, {
      center: [start.lat, start.lng],
      zoom: initial ? PIN_ZOOM : WORLD_ZOOM,
    });
    L.tileLayer(OSM_TILE_URL, { attribution: OSM_ATTRIBUTION }).addTo(map);

    function placeMarker(coords: Coordinates) {
      if (markerRef.current) {
        markerRef.current.setLatLng([coords.lat, coords.lng]);
        return;
      }
      const marker = L.marker([coords.lat, coords.lng], {
        draggable: true,
      }).addTo(map);
      marker.on("dragend", () => {
        const pos = marker.getLatLng();
        setSelected({ lat: pos.lat, lng: pos.lng });
      });
      markerRef.current = marker;
    }

    if (initial) placeMarker(initial);

    map.on("click", (e: L.LeafletMouseEvent) => {
      const coords = { lat: e.latlng.lat, lng: e.latlng.lng };
      placeMarker(coords);
      setSelected(coords);
    });

    const raf = requestAnimationFrame(() => map.invalidateSize());

    if (!initial && navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          if (cancelled) return;
          map.setView([pos.coords.latitude, pos.coords.longitude], PIN_ZOOM);
        },
        () => {
          // Denied/unavailable — the world view already showing is the
          // documented fallback, so this is deliberately a no-op.
        },
        { timeout: 5000 },
      );
    }

    return () => {
      cancelled = true;
      cancelAnimationFrame(raf);
      map.remove();
      markerRef.current = null;
    };
  }, [open, container, initial]);

  return (
    <Dialog open={open} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="flex w-full max-w-md flex-col gap-3 rounded-lg bg-white p-4 dark:bg-neutral-900">
          <DialogTitle className="text-base font-semibold">
            {t("entries.location.pickTitle")}
          </DialogTitle>
          <div
            ref={setContainer}
            className="h-72 w-full overflow-hidden rounded-md"
          />
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            {t("entries.location.pickHint")}
          </p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
            >
              {t("entries.location.cancel")}
            </button>
            <button
              type="button"
              disabled={!selected}
              onClick={() => selected && onConfirm(formatCoordinates(selected))}
              className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              {t("entries.location.confirm")}
            </button>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  );
}
