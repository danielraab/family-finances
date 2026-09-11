import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../api/schema";
import { TagLabel } from "./TagLabel";

type Tag = components["schemas"]["Tag"];

const SUGGESTION_LIMIT = 8;

/**
 * A free-text tag input: matches against the visitor's existing tags
 * (including ones shared with the visitor, via `existingTags`) as they
 * type, and lets them add any typed name — matched or not. Resolving an
 * unmatched name to a newly-created tag happens on form submit (see
 * entries.new.tsx / entries.$entryId.edit.tsx), not here — this component
 * only manages the set of tag *names* currently attached.
 *
 * Suggestions show as soon as the field is focused, even before typing —
 * an empty draft matches every not-yet-attached existing tag rather than
 * none, so a caller doesn't need to already know a tag's name to find it.
 */
export function TagInput({
  value,
  onChange,
  existingTags,
}: {
  value: string[];
  onChange: (names: string[]) => void;
  existingTags: Tag[];
}) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("");
  const [focused, setFocused] = useState(false);

  const byName = new Map(existingTags.map((tag) => [tag.name, tag]));

  const query = draft.trim().toLowerCase();
  const suggestions = existingTags
    .filter(
      (tag) =>
        !value.includes(tag.name) &&
        (!query || tag.name.toLowerCase().includes(query)),
    )
    .slice(0, SUGGESTION_LIMIT);
  const showSuggestions = focused && suggestions.length > 0;

  function addTag(name: string) {
    const trimmed = name.trim();
    if (!trimmed || value.includes(trimmed)) {
      setDraft("");
      return;
    }
    onChange([...value, trimmed]);
    setDraft("");
  }

  function removeTag(name: string) {
    onChange(value.filter((v) => v !== name));
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-1.5 rounded-md border border-black/15 bg-transparent px-2 py-1.5 focus-within:border-black/40 dark:border-white/15 dark:focus-within:border-white/40">
        {value.map((name) => (
          <span
            key={name}
            className="flex items-center gap-1 rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium dark:bg-white/10"
          >
            {(() => {
              const tag = byName.get(name);
              return tag ? <TagLabel tag={tag} /> : name;
            })()}
            <button
              type="button"
              onClick={() => removeTag(name)}
              aria-label={t("entries.form.removeTag", { name })}
              className="text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              ×
            </button>
          </span>
        ))}
        <input
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === ",") {
              event.preventDefault();
              addTag(draft);
            } else if (
              event.key === "Backspace" &&
              draft === "" &&
              value.length > 0
            ) {
              const last = value[value.length - 1];
              if (last !== undefined) {
                removeTag(last);
              }
            }
          }}
          placeholder={t("entries.form.tagsPlaceholder")}
          className="min-w-24 flex-1 bg-transparent px-1 py-0.5 text-sm outline-none"
        />
      </div>
      {showSuggestions && (
        <div className="flex flex-wrap gap-1.5">
          {suggestions.map((tag) => (
            <button
              key={tag.id}
              type="button"
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => addTag(tag.name)}
              className="rounded-full border border-black/10 px-2 py-0.5 text-xs text-zinc-600 hover:bg-black/[.04] dark:border-white/10 dark:text-zinc-400 dark:hover:bg-white/[.06]"
            >
              <TagLabel tag={tag} />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
