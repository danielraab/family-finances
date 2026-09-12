import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import L from "leaflet";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Coordinates } from "../lib/location";
import { OSM_ATTRIBUTION, OSM_TILE_URL } from "./leafletSetup";

type LocationPreviewModalProps = {
  open: boolean;
  onClose: () => void;
  coordinates: Coordinates | null;
};

const PIN_ZOOM = 15;

// Read-only: a static marker centered on coordinates, no interaction — the
// ledger's globe icon opens this, distinct from LocationPickerModal's
// interactive, draggable-pin editing (see design.md's "two separate map
// surfaces" decision).
export function LocationPreviewModal({
  open,
  onClose,
  coordinates,
}: LocationPreviewModalProps) {
  const { t } = useTranslation();
  // A callback ref in state, not a plain useRef — see LocationPickerModal's
  // comment: Dialog's portal-mounted panel means a ref only reliably
  // becomes available in time to re-render (and re-run this effect) when
  // it's tracked as state.
  const [container, setContainer] = useState<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open || !coordinates || !container) return;
    const map = L.map(container, {
      center: [coordinates.lat, coordinates.lng],
      zoom: PIN_ZOOM,
      scrollWheelZoom: false,
    });
    L.tileLayer(OSM_TILE_URL, { attribution: OSM_ATTRIBUTION }).addTo(map);
    L.marker([coordinates.lat, coordinates.lng]).addTo(map);
    const raf = requestAnimationFrame(() => map.invalidateSize());
    return () => {
      cancelAnimationFrame(raf);
      map.remove();
    };
  }, [open, container, coordinates]);

  return (
    <Dialog open={open} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="flex w-full max-w-md flex-col gap-3 rounded-lg bg-white p-4 dark:bg-neutral-900">
          <DialogTitle className="text-base font-semibold">
            {t("entries.location.previewTitle")}
          </DialogTitle>
          <div
            ref={setContainer}
            className="h-72 w-full overflow-hidden rounded-md"
          />
          <div className="flex justify-end">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
            >
              {t("entries.location.close")}
            </button>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  );
}
