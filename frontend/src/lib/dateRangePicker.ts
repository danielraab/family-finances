import {
  CUSTOM_RANGE_KEY,
  type DateRangeValue,
  type EffectiveDateRange,
  type PresetKey,
} from "./dateRangePresets";

export type DateRangeDraft = {
  selectedKey: PresetKey | typeof CUSTOM_RANGE_KEY;
  from: string | undefined;
  to: string | undefined;
  phase: "idle" | "selecting_end";
};

export function parseLocalDate(value: string | undefined): Date | undefined {
  if (!value) return undefined;
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return undefined;
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const parsed = new Date(year, month - 1, day);
  if (
    parsed.getFullYear() !== year ||
    parsed.getMonth() !== month - 1 ||
    parsed.getDate() !== day
  ) {
    return undefined;
  }
  return parsed;
}

function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

export function toLocalDateString(date: Date): string {
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`;
}

export function formatLocalDate(
  value: string | undefined,
  language: string,
): string | undefined {
  const date = parseLocalDate(value);
  return date
    ? new Intl.DateTimeFormat(language, {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
      }).format(date)
    : value;
}

export function isValidDateRange(
  from: string | undefined,
  to: string | undefined,
): boolean {
  return from === undefined || to === undefined || from <= to;
}

export function createDateRangeDraft(
  effective: EffectiveDateRange,
): DateRangeDraft {
  return { ...effective, phase: "idle" };
}

export function selectCustomDay(
  draft: DateRangeDraft,
  date: Date,
): DateRangeDraft {
  const value = toLocalDateString(date);
  if (
    draft.selectedKey !== CUSTOM_RANGE_KEY ||
    draft.phase === "idle" ||
    draft.from === undefined ||
    draft.to !== undefined
  ) {
    return {
      selectedKey: CUSTOM_RANGE_KEY,
      from: value,
      to: undefined,
      phase: "selecting_end",
    };
  }
  if (value < draft.from) {
    return { ...draft, from: value, to: undefined };
  }
  return { ...draft, to: value, phase: "idle" };
}

export function selectStartOnly(draft: DateRangeDraft): DateRangeDraft {
  const from = draft.from ?? draft.to;
  return {
    selectedKey: CUSTOM_RANGE_KEY,
    from,
    to: undefined,
    phase: "idle",
  };
}

export function selectEndOnly(draft: DateRangeDraft): DateRangeDraft {
  const to = draft.to ?? draft.from;
  return {
    selectedKey: CUSTOM_RANGE_KEY,
    from: undefined,
    to,
    phase: "idle",
  };
}

export function dateRangeDraftToPatch(
  draft: DateRangeDraft,
): DateRangeValue | undefined {
  if (!isValidDateRange(draft.from, draft.to)) return undefined;
  if (draft.selectedKey !== CUSTOM_RANGE_KEY) {
    return { range: draft.selectedKey, from: undefined, to: undefined };
  }
  if (draft.from === undefined && draft.to === undefined) {
    return { range: CUSTOM_RANGE_KEY, from: undefined, to: undefined };
  }
  return {
    range: undefined,
    from: draft.from,
    to: draft.to,
  };
}
