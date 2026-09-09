import {
  CloseButton,
  Popover,
  PopoverButton,
  PopoverPanel,
} from "@headlessui/react";
import { ChevronDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ENTITY_COLOR_TOKENS, entityColorVar } from "../lib/entityColors";
import { ENTITY_ICON_GROUPS } from "../lib/entityIcons";
import { EntityIcon } from "./EntityIcon";

export type IconColorValue = { icon: string; color: string };

type IconColorPickerProps = {
  value: IconColorValue;
  onChange: (value: IconColorValue) => void;
};

const cellBase =
  "flex items-center justify-center rounded-md border transition-colors";
const cellIdle =
  "border-black/10 hover:border-black/30 dark:border-white/10 dark:hover:border-white/30";
const cellActive = "border-black/70 dark:border-white/70";

/**
 * Independent icon and colour pickers for an account or category. Either,
 * both, or neither may be set; an empty string means unset. A trigger
 * button shows the current selection and opens a popover holding the
 * colour swatches and the grouped icon grid. Used by the account form and
 * the category create/rename forms.
 */
export function IconColorPicker({ value, onChange }: IconColorPickerProps) {
  const { t } = useTranslation();

  const setIcon = (icon: string) =>
    onChange({ ...value, icon: value.icon === icon ? "" : icon });
  const setColor = (color: string) =>
    onChange({ ...value, color: value.color === color ? "" : color });

  const hasSelection = value.icon !== "" || value.color !== "";

  return (
    <div className="flex flex-col gap-1.5 text-sm font-medium">
      {t("entityIcons.fieldLabel")}
      <Popover className="relative">
        <PopoverButton className="flex items-center gap-2 rounded-md border border-black/15 bg-transparent px-3 py-2 text-left text-sm font-normal outline-none transition-colors focus:border-black/40 data-[open]:border-black/40 dark:border-white/15 dark:focus:border-white/40 dark:data-[open]:border-white/40">
          {hasSelection ? (
            <EntityIcon icon={value.icon} color={value.color} size={20} />
          ) : (
            <span className="size-5 rounded-[5px] border border-dashed border-black/25 dark:border-white/25" />
          )}
          <span className="flex-1 text-zinc-600 dark:text-zinc-400">
            {hasSelection
              ? [value.icon, value.color].filter(Boolean).join(" · ")
              : t("entityIcons.placeholder")}
          </span>
          <ChevronDown
            size={16}
            className="text-zinc-400 dark:text-zinc-500"
            aria-hidden="true"
          />
        </PopoverButton>

        <PopoverPanel
          anchor="bottom start"
          className="z-[60] mt-1 flex w-[19rem] flex-col gap-3 rounded-lg border border-black/10 bg-white p-3 text-sm font-normal shadow-lg dark:border-white/15 dark:bg-neutral-900"
        >
          <div className="flex items-center justify-between">
            <span className="font-medium">{t("entityIcons.colorLabel")}</span>
            <CloseButton className="rounded-md px-2 py-0.5 text-xs text-zinc-500 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]">
              {t("entityIcons.done")}
            </CloseButton>
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            <button
              type="button"
              onClick={() => onChange({ ...value, color: "" })}
              className={`${cellBase} h-7 px-2 text-xs ${
                value.color === "" ? cellActive : cellIdle
              }`}
              aria-pressed={value.color === ""}
            >
              {t("entityIcons.none")}
            </button>
            {ENTITY_COLOR_TOKENS.map((token) => (
              <button
                key={token}
                type="button"
                onClick={() => setColor(token)}
                title={token}
                aria-label={token}
                aria-pressed={value.color === token}
                className={`${cellBase} size-7 ${
                  value.color === token ? cellActive : cellIdle
                }`}
              >
                <span
                  className="size-4 rounded-full"
                  style={{ backgroundColor: entityColorVar(token) }}
                />
              </button>
            ))}
          </div>

          <span className="font-medium">{t("entityIcons.iconLabel")}</span>
          <div className="flex max-h-56 flex-col gap-2 overflow-y-auto rounded-md border border-black/10 p-2 dark:border-white/10">
            <button
              type="button"
              onClick={() => onChange({ ...value, icon: "" })}
              className={`${cellBase} h-7 self-start px-2 text-xs ${
                value.icon === "" ? cellActive : cellIdle
              }`}
              aria-pressed={value.icon === ""}
            >
              {t("entityIcons.none")}
            </button>
            {ENTITY_ICON_GROUPS.map((group) => (
              <div key={group.headingKey} className="flex flex-col gap-1">
                <span className="text-xs text-black/50 dark:text-white/50">
                  {t(group.headingKey)}
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {group.icons.map(({ token }) => (
                    <button
                      key={token}
                      type="button"
                      onClick={() => setIcon(token)}
                      title={token}
                      aria-label={token}
                      aria-pressed={value.icon === token}
                      className={`${cellBase} size-8 ${
                        value.icon === token ? cellActive : cellIdle
                      }`}
                    >
                      <EntityIcon icon={token} size={18} />
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </PopoverPanel>
      </Popover>
    </div>
  );
}
