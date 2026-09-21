import { type DateRange, DayPicker } from "@daypicker/react";
import { de, enUS } from "@daypicker/react/locale";
import "@daypicker/react/style.css";
import {
  Dialog,
  DialogBackdrop,
  DialogPanel,
  Popover,
  PopoverButton,
  PopoverPanel,
} from "@headlessui/react";
import { CalendarDays, ChevronDown, Info, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  createDateRangeDraft,
  type DateRangeDraft,
  dateRangeDraftToPatch,
  formatLocalDate,
  isValidDateRange,
  parseLocalDate,
  selectCustomDay,
  selectEndOnly,
  selectStartOnly,
} from "../lib/dateRangePicker";
import {
  ALL_TIME_KEY,
  CUSTOM_RANGE_KEY,
  DATE_RANGE_PRESET_KEYS,
  type DateRangeValue,
  PRESET_I18N_KEYS,
  type PresetKey,
  resolveEffectiveRange,
  resolvePreset,
  type WeekStart,
} from "../lib/dateRangePresets";

const triggerClass =
  "rounded-md border border-black/15 bg-transparent px-2.5 py-1.5 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

type DateRangeFilterProps = {
  value: DateRangeValue;
  weekStart: WeekStart;
  defaultPreset?: PresetKey | undefined;
  onChange: (patch: DateRangeValue) => void;
  fromLabel: string;
  toLabel: string;
  fullWidth?: boolean | undefined;
};

function useMobilePicker(): boolean {
  const [mobile, setMobile] = useState(
    () =>
      typeof window !== "undefined" &&
      !window.matchMedia("(min-width: 640px)").matches,
  );
  useEffect(() => {
    const query = window.matchMedia("(min-width: 640px)");
    const update = () => setMobile(!query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return mobile;
}

function useTwoMonthPicker(): boolean {
  const [twoMonths, setTwoMonths] = useState(
    () =>
      typeof window !== "undefined" &&
      window.matchMedia("(min-width: 900px)").matches,
  );
  useEffect(() => {
    const query = window.matchMedia("(min-width: 900px)");
    const update = () => setTwoMonths(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return twoMonths;
}

function summaryFor(
  draft: DateRangeDraft,
  language: string,
  t: ReturnType<typeof useTranslation>["t"],
): string {
  if (draft.selectedKey !== CUSTOM_RANGE_KEY) {
    return t(`dateRangeFilter.presets.${PRESET_I18N_KEYS[draft.selectedKey]}`);
  }
  const from = formatLocalDate(draft.from, language);
  const to = formatLocalDate(draft.to, language);
  if (from && to) return `${from} – ${to}`;
  if (from) return t("dateRangeFilter.fromOnly", { date: from });
  if (to) return t("dateRangeFilter.toOnly", { date: to });
  return t("dateRangeFilter.placeholder");
}

type PickerContentProps = {
  draft: DateRangeDraft;
  setDraft: (draft: DateRangeDraft) => void;
  weekStart: WeekStart;
  numberOfMonths: number;
  onApply: () => void;
  onClose: () => void;
};

function PickerContent({
  draft,
  setDraft,
  weekStart,
  numberOfMonths,
  onApply,
  onClose,
}: PickerContentProps) {
  const { t, i18n } = useTranslation();
  const language = i18n.resolvedLanguage ?? i18n.language;
  const fromDate = parseLocalDate(draft.from);
  const toDate = parseLocalDate(draft.to);
  const valid = isValidDateRange(draft.from, draft.to);
  const canApply = valid && draft.phase !== "selecting_end";
  const selected: DateRange | undefined =
    valid && (fromDate || toDate)
      ? { from: fromDate ?? toDate, to: toDate ?? fromDate }
      : undefined;
  const defaultMonth = fromDate ?? toDate ?? new Date();

  function selectPreset(key: PresetKey) {
    const range = resolvePreset(key, weekStart, new Date());
    setDraft({ selectedKey: key, ...range, phase: "idle" });
  }

  return (
    <div className="flex min-h-0 flex-col bg-white text-zinc-900 dark:bg-neutral-900 dark:text-zinc-100">
      <div className="flex items-center justify-between gap-3 border-b border-black/10 px-4 py-3 dark:border-white/10">
        <h2 className="text-base font-semibold sm:text-sm">
          {t("dateRangeFilter.label")}
        </h2>
        <div className="flex items-center gap-1">
          <button
            type="button"
            className="min-h-11 rounded-md px-3 text-sm text-indigo-600 hover:bg-indigo-50 sm:min-h-0 sm:py-1.5 dark:text-indigo-300 dark:hover:bg-indigo-400/10"
            onClick={() => selectPreset(ALL_TIME_KEY)}
          >
            {t("dateRangeFilter.reset")}
          </button>
          <button
            type="button"
            className="grid size-11 place-items-center rounded-md hover:bg-black/[.04] sm:size-8 dark:hover:bg-white/[.06]"
            onClick={onClose}
            aria-label={t("dateRangeFilter.close")}
          >
            <X size={18} aria-hidden="true" />
          </button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <div className="-mx-4 flex snap-x gap-2 overflow-x-auto px-4 pb-2 sm:mx-0 sm:grid sm:grid-cols-3 sm:overflow-visible sm:px-0">
          {DATE_RANGE_PRESET_KEYS.map((key) => {
            const active = draft.selectedKey === key;
            return (
              <button
                key={key}
                type="button"
                className={`min-h-11 shrink-0 snap-start rounded-md border px-3 py-2 text-sm transition-colors sm:min-h-0 ${
                  active
                    ? "border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-200"
                    : "border-black/10 hover:bg-black/[.03] dark:border-white/15 dark:hover:bg-white/[.05]"
                }`}
                aria-pressed={active}
                onClick={() => selectPreset(key)}
              >
                {t(`dateRangeFilter.presets.${PRESET_I18N_KEYS[key]}`)}
              </button>
            );
          })}
        </div>

        <div className="mt-4 border-t border-black/10 pt-4 dark:border-white/10">
          <h3 className="text-sm font-semibold">
            {t("dateRangeFilter.presets.custom")}
          </h3>
          <div className="mt-2 flex min-h-11 items-center gap-2 rounded-md border border-black/15 px-3 text-sm dark:border-white/15">
            <CalendarDays
              size={16}
              className="text-zinc-500"
              aria-hidden="true"
            />
            {summaryFor(draft, language, t)}
          </div>

          <DayPicker
            key={`${numberOfMonths}-${defaultMonth.getFullYear()}-${defaultMonth.getMonth()}`}
            mode="range"
            selected={selected}
            defaultMonth={defaultMonth}
            numberOfMonths={numberOfMonths}
            pagedNavigation={numberOfMonths > 1}
            showOutsideDays
            weekStartsOn={weekStart === "monday" ? 1 : 0}
            locale={language.startsWith("de") ? de : enUS}
            onDayClick={(day) => setDraft(selectCustomDay(draft, day))}
            labels={{
              labelPrevious: () => t("dateRangeFilter.previousMonth"),
              labelNext: () => t("dateRangeFilter.nextMonth"),
            }}
            className="date-range-calendar mx-auto mt-3"
          />

          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <button
              type="button"
              className="min-h-11 rounded-md border border-black/15 px-3 text-sm text-indigo-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/15 dark:text-indigo-300"
              disabled={!draft.from && !draft.to}
              onClick={() => setDraft(selectStartOnly(draft))}
            >
              {t("dateRangeFilter.useStartOnly")}
            </button>
            <button
              type="button"
              className="min-h-11 rounded-md border border-black/15 px-3 text-sm text-indigo-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/15 dark:text-indigo-300"
              disabled={!draft.from && !draft.to}
              onClick={() => setDraft(selectEndOnly(draft))}
            >
              {t("dateRangeFilter.useEndOnly")}
            </button>
          </div>

          <p
            className={`mt-3 flex items-start gap-2 text-xs ${
              valid
                ? "text-zinc-500 dark:text-zinc-400"
                : "text-red-600 dark:text-red-400"
            }`}
            role={valid ? undefined : "alert"}
          >
            <Info size={15} className="mt-0.5 shrink-0" aria-hidden="true" />
            {valid
              ? t("dateRangeFilter.guidance")
              : t("dateRangeFilter.invalidOrder")}
          </p>
        </div>
      </div>

      <div className="border-t border-black/10 px-4 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] dark:border-white/10">
        <button
          type="button"
          className="min-h-11 w-full rounded-md bg-indigo-600 px-4 text-sm font-medium text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          disabled={!canApply}
          onClick={onApply}
        >
          {t("dateRangeFilter.apply")}
        </button>
      </div>
    </div>
  );
}

export function DateRangeFilter({
  value,
  weekStart,
  defaultPreset,
  onChange,
  fullWidth = false,
}: DateRangeFilterProps) {
  const { t, i18n } = useTranslation();
  const mobile = useMobilePicker();
  const twoMonths = useTwoMonthPicker();
  const effective = resolveEffectiveRange(
    value,
    weekStart,
    defaultPreset,
    new Date(),
  );
  const [draft, setDraft] = useState(() => createDateRangeDraft(effective));
  const [mobileOpen, setMobileOpen] = useState(false);
  const language = i18n.resolvedLanguage ?? i18n.language;
  const summary = summaryFor(createDateRangeDraft(effective), language, t);

  function openDraft() {
    setDraft(createDateRangeDraft(effective));
  }

  function apply(close: () => void) {
    const patch = dateRangeDraftToPatch(draft);
    if (!patch) return;
    onChange(patch);
    close();
  }

  const trigger = (
    <>
      <span className="flex-1 truncate text-zinc-900 dark:text-zinc-100">
        {summary}
      </span>
      <ChevronDown
        size={14}
        className="shrink-0 text-zinc-400"
        aria-hidden="true"
      />
    </>
  );

  return (
    <div className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
      {t("dateRangeFilter.label")}
      {mobile ? (
        <>
          <button
            type="button"
            className={`${triggerClass} flex min-w-40 items-center gap-2 text-left ${fullWidth ? "w-full" : ""}`}
            onClick={() => {
              openDraft();
              setMobileOpen(true);
            }}
          >
            {trigger}
          </button>
          <Dialog
            open={mobileOpen}
            onClose={() => setMobileOpen(false)}
            className="relative z-[70]"
            aria-label={t("dateRangeFilter.label")}
          >
            <DialogBackdrop className="fixed inset-0 bg-black/45" />
            <div className="fixed inset-0 flex items-end">
              <DialogPanel className="max-h-[92dvh] w-full overflow-hidden rounded-t-2xl shadow-2xl">
                <div className="mx-auto -mb-2 mt-2 h-1.5 w-12 rounded-full bg-zinc-300 dark:bg-zinc-600" />
                <PickerContent
                  draft={draft}
                  setDraft={setDraft}
                  weekStart={weekStart}
                  numberOfMonths={1}
                  onClose={() => setMobileOpen(false)}
                  onApply={() => apply(() => setMobileOpen(false))}
                />
              </DialogPanel>
            </div>
          </Dialog>
        </>
      ) : (
        <Popover className="relative">
          {({ close }) => (
            <>
              <PopoverButton
                className={`${triggerClass} flex min-w-40 items-center gap-2 text-left data-[open]:border-black/40 dark:data-[open]:border-white/40 ${fullWidth ? "w-full" : ""}`}
                onClick={openDraft}
              >
                {trigger}
              </PopoverButton>
              <PopoverPanel
                anchor="bottom start"
                className="z-[60] mt-1 w-[min(42rem,calc(100vw-2rem))] overflow-hidden rounded-lg border border-black/10 shadow-xl dark:border-white/15"
              >
                <PickerContent
                  draft={draft}
                  setDraft={setDraft}
                  weekStart={weekStart}
                  numberOfMonths={twoMonths ? 2 : 1}
                  onClose={close}
                  onApply={() => apply(close)}
                />
              </PopoverPanel>
            </>
          )}
        </Popover>
      )}
    </div>
  );
}
