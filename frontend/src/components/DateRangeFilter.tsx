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
const TWO_MONTH_PICKER_QUERY = "(min-width: 960px)";

// @daypicker/react's range-selection hook only reads `selected` as a
// controlled prop when `onSelect` is also supplied — without it, DayPicker
// seeds its display from `selected` once at mount and then tracks its own
// internal click state, silently diverging from `draft` (e.g. clicking "Use
// start only" would update the summary but leave the old full range
// highlighted). Selection itself is driven entirely through `onDayClick`
// below, so this callback intentionally does nothing.
function noopOnSelect() {}

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
      window.matchMedia(TWO_MONTH_PICKER_QUERY).matches,
  );
  useEffect(() => {
    const query = window.matchMedia(TWO_MONTH_PICKER_QUERY);
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
  // `to`/`from` are left genuinely undefined (rather than falling back to
  // the other bound) so an open-ended draft — e.g. after "Use start only" —
  // highlights only its one defined bound, not a fake single-day range.
  const selected: DateRange | undefined =
    valid && (fromDate || toDate) ? { from: fromDate, to: toDate } : undefined;
  const defaultMonth = fromDate ?? toDate ?? new Date();

  function selectPreset(key: PresetKey) {
    const range = resolvePreset(key, weekStart, new Date());
    setDraft({ selectedKey: key, ...range, phase: "idle" });
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col bg-white text-zinc-900 dark:bg-neutral-900 dark:text-zinc-100">
      <div className="flex items-center justify-between gap-3 border-b border-black/10 px-4 py-3 sm:px-3 sm:py-2 dark:border-white/10">
        <h2 className="text-base font-semibold sm:text-sm">
          {t("dateRangeFilter.label")}
        </h2>
        <div className="flex items-center gap-1">
          <button
            type="button"
            className="min-h-11 rounded-md px-3 text-sm text-zinc-600 hover:bg-black/[.04] sm:min-h-0 sm:px-2 sm:py-1 sm:text-xs dark:text-zinc-300 dark:hover:bg-white/[.06]"
            onClick={() => selectPreset(ALL_TIME_KEY)}
          >
            {t("dateRangeFilter.reset")}
          </button>
          <button
            type="button"
            className="grid size-11 place-items-center rounded-md hover:bg-black/[.04] sm:size-7 dark:hover:bg-white/[.06]"
            onClick={onClose}
            aria-label={t("dateRangeFilter.close")}
          >
            <X size={16} aria-hidden="true" />
          </button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4 sm:px-3 sm:py-3">
        <div className="-mx-4 flex snap-x gap-2 overflow-x-auto px-4 pb-2 sm:mx-0 sm:grid sm:grid-cols-3 sm:gap-1.5 sm:overflow-visible sm:px-0 sm:pb-0">
          {DATE_RANGE_PRESET_KEYS.map((key) => {
            const active = draft.selectedKey === key;
            return (
              <button
                key={key}
                type="button"
                className={`min-h-11 shrink-0 snap-start rounded-md border px-3 py-2 text-sm transition-colors sm:min-h-0 sm:px-2.5 sm:py-1 sm:text-xs ${
                  active
                    ? "border-black/40 bg-black/[.06] text-black dark:border-white/40 dark:bg-white/10 dark:text-white"
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

        <div className="mt-4 border-t border-black/10 pt-4 sm:mt-3 sm:pt-3 dark:border-white/10">
          <h3 className="text-sm font-semibold sm:text-xs sm:font-medium sm:uppercase sm:tracking-wide sm:text-zinc-500 sm:dark:text-zinc-400">
            {t("dateRangeFilter.presets.custom")}
          </h3>
          <div className="mt-2 flex min-h-11 items-center gap-2 rounded-md border border-black/15 px-3 text-sm sm:min-h-0 sm:gap-1.5 sm:py-1.5 sm:text-xs dark:border-white/15">
            <CalendarDays
              size={14}
              className="shrink-0 text-zinc-500"
              aria-hidden="true"
            />
            <span className="flex-1 truncate">
              {summaryFor(draft, language, t)}
            </span>
            <button
              type="button"
              className="hidden shrink-0 rounded border border-black/15 px-1.5 py-0.5 text-xs text-zinc-600 transition-colors hover:bg-black/[.04] disabled:cursor-not-allowed disabled:opacity-40 sm:inline-flex dark:border-white/15 dark:text-zinc-300 dark:hover:bg-white/[.06]"
              disabled={!draft.from && !draft.to}
              onClick={() => setDraft(selectStartOnly(draft))}
            >
              {t("dateRangeFilter.useStartOnly")}
            </button>
            <button
              type="button"
              className="hidden shrink-0 rounded border border-black/15 px-1.5 py-0.5 text-xs text-zinc-600 transition-colors hover:bg-black/[.04] disabled:cursor-not-allowed disabled:opacity-40 sm:inline-flex dark:border-white/15 dark:text-zinc-300 dark:hover:bg-white/[.06]"
              disabled={!draft.from && !draft.to}
              onClick={() => setDraft(selectEndOnly(draft))}
            >
              {t("dateRangeFilter.useEndOnly")}
            </button>
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
            onSelect={noopOnSelect}
            labels={{
              labelPrevious: () => t("dateRangeFilter.previousMonth"),
              labelNext: () => t("dateRangeFilter.nextMonth"),
            }}
            className="date-range-calendar mx-auto mt-3 sm:mt-2"
          />

          {/* @daypicker/react's mobile bottom sheet still needs these as
              full-size 44px touch targets (design.md decision 8); the sm+
              popover offers them as the inline chips in the summary bar
              above instead, so this block is mobile-only. */}
          <div className="mt-3 grid gap-2 sm:hidden">
            <button
              type="button"
              className="min-h-11 rounded-md border border-black/15 px-3 text-sm text-zinc-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/15 dark:text-zinc-300"
              disabled={!draft.from && !draft.to}
              onClick={() => setDraft(selectStartOnly(draft))}
            >
              {t("dateRangeFilter.useStartOnly")}
            </button>
            <button
              type="button"
              className="min-h-11 rounded-md border border-black/15 px-3 text-sm text-zinc-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/15 dark:text-zinc-300"
              disabled={!draft.from && !draft.to}
              onClick={() => setDraft(selectEndOnly(draft))}
            >
              {t("dateRangeFilter.useEndOnly")}
            </button>
          </div>

          <p
            className={`mt-3 flex items-start gap-2 text-xs sm:mt-2 ${
              valid
                ? "text-zinc-500 dark:text-zinc-400"
                : "text-red-600 dark:text-red-400"
            }`}
            role={valid ? undefined : "alert"}
          >
            <Info size={14} className="mt-0.5 shrink-0" aria-hidden="true" />
            {valid
              ? t("dateRangeFilter.guidance")
              : t("dateRangeFilter.invalidOrder")}
          </p>
        </div>
      </div>

      <div className="border-t border-black/10 px-4 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] sm:px-3 sm:pt-2.5 sm:pb-2.5 dark:border-white/10">
        <button
          type="button"
          className="min-h-11 w-full rounded-md bg-zinc-900 px-4 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:cursor-not-allowed disabled:opacity-60 sm:min-h-0 sm:py-2 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
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
              <DialogPanel className="flex max-h-[92dvh] w-full flex-col overflow-hidden rounded-t-2xl shadow-2xl">
                <div className="mx-auto -mb-2 mt-2 h-1.5 w-12 shrink-0 rounded-full bg-zinc-300 dark:bg-zinc-600" />
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
                className="z-[60] mt-1 w-[min(31rem,calc(100vw-2rem))] overflow-hidden rounded-lg border border-black/10 shadow-xl dark:border-white/15"
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
