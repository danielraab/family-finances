import { useState } from "react";
import { useTranslation } from "react-i18next";
import { formatCoordinates, parseLocation } from "../lib/location";
import { LocationPickerModal } from "./LocationPickerModal";

type LocationFieldProps = {
  value: string;
  onChange: (value: string) => void;
  inputClassName: string;
};

const buttonClass =
  "whitespace-nowrap rounded-md border border-black/15 px-2.5 py-1.5 text-xs font-medium text-zinc-600 transition-colors hover:bg-black/[.04] dark:border-white/15 dark:text-zinc-400 dark:hover:bg-white/[.06]";

// The field always shows its literal stored value, including raw JSON
// coordinates when that's what GPS/the map picker wrote — no separate
// prettified display (see design.md's "always shows its raw value"
// decision).
export function LocationField({
  value,
  onChange,
  inputClassName,
}: LocationFieldProps) {
  const { t } = useTranslation();
  const [pickerOpen, setPickerOpen] = useState(false);
  const [gpsError, setGpsError] = useState<string | null>(null);

  function useGps() {
    setGpsError(null);
    if (!navigator.geolocation) {
      setGpsError(t("entries.form.locationGpsUnsupported"));
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        onChange(
          formatCoordinates({
            lat: pos.coords.latitude,
            lng: pos.coords.longitude,
          }),
        );
      },
      () => {
        setGpsError(t("entries.form.locationGpsError"));
      },
      { timeout: 10_000 },
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap gap-2">
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={t("entries.form.locationPlaceholder")}
          className={`min-w-0 flex-1 ${inputClassName}`}
        />
        <button type="button" onClick={useGps} className={buttonClass}>
          {t("entries.form.locationUseGps")}
        </button>
        <button
          type="button"
          onClick={() => setPickerOpen(true)}
          className={buttonClass}
        >
          {t("entries.form.locationPickOnMap")}
        </button>
      </div>
      {gpsError && (
        <span className="text-xs font-normal text-red-600 dark:text-red-400">
          {gpsError}
        </span>
      )}
      <LocationPickerModal
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        initial={parseLocation(value)}
        onConfirm={(next) => {
          onChange(next);
          setPickerOpen(false);
        }}
      />
    </div>
  );
}
