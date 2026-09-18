import {
  CloseButton,
  Popover,
  PopoverButton,
  PopoverPanel,
} from "@headlessui/react";
import { ChevronDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  CUSTOM_RANGE_KEY,
  DATE_RANGE_PRESET_KEYS,
  type DateRangeValue,
  PRESET_I18N_KEYS,
  type PresetKey,
  resolveEffectiveRange,
  type WeekStart,
} from "../lib/dateRangePresets";

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-2.5 py-1.5 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40 disabled:opacity-60";

type DateRangeFilterProps = {
  value: DateRangeValue;
  weekStart: WeekStart;
  defaultPreset?: PresetKey | undefined;
  onChange: (patch: DateRangeValue) => void;
  fromLabel: string;
  toLabel: string;
  /** Stretch the trigger to its container's width instead of sizing it to
   * the summary text — for a caller laying its filters out in a grid,
   * where a content-width trigger is the odd one out beside the
   * full-width `select`s. Off by default, so the flex-wrap filter rows
   * that predate the grid keep their content-width trigger. */
  fullWidth?: boolean | undefined;
};

/**
 * The shared date-range filter: a single trigger button summarizing the
 * current selection, opening a popover with the named-preset dropdown and
 * the two date inputs — the two inputs stay visible inside the popover
 * under every mode, populated and disabled while a preset is active,
 * editable under Custom. See openspec/specs/web-client-date-range-filter
 * for the full contract. Collapsed to one trigger (rather than three
 * always-visible fields) so it doesn't dominate a filter row the way three
 * separate controls did.
 */
export function DateRangeFilter({
  value,
  weekStart,
  defaultPreset,
  onChange,
  fromLabel,
  toLabel,
  fullWidth = false,
}: DateRangeFilterProps) {
  const { t } = useTranslation();
  const today = new Date();
  const effective = resolveEffectiveRange(
    value,
    weekStart,
    defaultPreset,
    today,
  );
  const isCustom = effective.selectedKey === CUSTOM_RANGE_KEY;

  function selectPreset(key: string) {
    if (key === CUSTOM_RANGE_KEY) {
      onChange({ range: undefined, from: effective.from, to: effective.to });
    } else {
      onChange({ range: key, from: undefined, to: undefined });
    }
  }

  function changeFrom(newValue: string) {
    if (isCustom) {
      onChange({ from: newValue || undefined });
    } else {
      onChange({
        range: undefined,
        from: newValue || undefined,
        to: effective.to,
      });
    }
  }

  function changeTo(newValue: string) {
    if (isCustom) {
      onChange({ to: newValue || undefined });
    } else {
      onChange({
        range: undefined,
        from: effective.from,
        to: newValue || undefined,
      });
    }
  }

  const selectedPreset =
    effective.selectedKey !== CUSTOM_RANGE_KEY ? effective.selectedKey : null;
  const summary = selectedPreset
    ? t(`dateRangeFilter.presets.${PRESET_I18N_KEYS[selectedPreset]}`)
    : effective.from && effective.to
      ? `${effective.from} – ${effective.to}`
      : effective.from
        ? t("dateRangeFilter.fromOnly", { date: effective.from })
        : effective.to
          ? t("dateRangeFilter.toOnly", { date: effective.to })
          : t("dateRangeFilter.placeholder");

  return (
    <div className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
      {t("dateRangeFilter.label")}
      <Popover className="relative">
        <PopoverButton
          className={`${inputClass} flex min-w-40 items-center gap-2 text-left data-[open]:border-black/40 dark:data-[open]:border-white/40 ${fullWidth ? "w-full" : ""}`}
        >
          <span className="flex-1 truncate text-zinc-900 dark:text-zinc-100">
            {summary}
          </span>
          <ChevronDown
            size={14}
            className="shrink-0 text-zinc-400 dark:text-zinc-500"
            aria-hidden="true"
          />
        </PopoverButton>

        <PopoverPanel
          anchor="bottom start"
          className="z-[60] mt-1 flex w-72 flex-col gap-3 rounded-lg border border-black/10 bg-white p-3 text-sm font-normal shadow-lg dark:border-white/15 dark:bg-neutral-900"
        >
          <div className="flex items-center justify-between">
            <span className="font-medium">{t("dateRangeFilter.label")}</span>
            <CloseButton className="rounded-md px-2 py-0.5 text-xs text-zinc-500 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]">
              {t("dateRangeFilter.done")}
            </CloseButton>
          </div>

          <select
            className={inputClass}
            value={effective.selectedKey}
            onChange={(e) => selectPreset(e.target.value)}
          >
            {DATE_RANGE_PRESET_KEYS.map((key) => (
              <option key={key} value={key}>
                {t(`dateRangeFilter.presets.${PRESET_I18N_KEYS[key]}`)}
              </option>
            ))}
            <option value={CUSTOM_RANGE_KEY}>
              {t("dateRangeFilter.presets.custom")}
            </option>
          </select>

          <div className="grid grid-cols-2 gap-2">
            <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
              {fromLabel}
              <input
                type="date"
                className={inputClass}
                value={effective.from ?? ""}
                disabled={!isCustom}
                onChange={(e) => changeFrom(e.target.value)}
              />
            </label>

            <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
              {toLabel}
              <input
                type="date"
                className={inputClass}
                value={effective.to ?? ""}
                disabled={!isCustom}
                onChange={(e) => changeTo(e.target.value)}
              />
            </label>
          </div>
        </PopoverPanel>
      </Popover>
    </div>
  );
}
