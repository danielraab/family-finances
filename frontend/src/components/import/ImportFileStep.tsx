import { type ChangeEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  isParseFileError,
  type ParsedFile,
  parseImportFile,
} from "../../lib/import/parseFile";

/**
 * Step 2: select a `.csv` or `.json` file and parse it entirely
 * client-side. The CSV-header-row note is always visible, not only on
 * error — see `web-client-entry-import`'s "A CSV or JSON file is selected
 * and parsed client-side" requirement.
 */
export function ImportFileStep({
  onParsed,
  onBack,
}: {
  onParsed: (parsed: ParsedFile) => void;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  const [parsing, setParsing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fileName, setFileName] = useState<string | null>(null);

  async function handleFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    setFileName(file.name);
    setError(null);
    setParsing(true);
    const result = await parseImportFile(file);
    setParsing(false);

    if (isParseFileError(result)) {
      setError(result.message);
      return;
    }
    onParsed(result);
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        {t("entries.import.steps.file.csvHeaderNote")}
      </p>

      <div className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.import.steps.file.label")}
        <div className="flex flex-wrap items-center gap-3">
          <label className="cursor-pointer rounded-md border border-black/15 px-4 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]">
            {t("entries.import.steps.file.chooseFile")}
            <input
              type="file"
              accept=".csv,.json"
              onChange={handleFile}
              className="sr-only"
            />
          </label>
          <span className="text-sm font-normal text-zinc-600 dark:text-zinc-400">
            {fileName ?? t("entries.import.steps.file.noFileChosen")}
          </span>
        </div>
      </div>

      {parsing && (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("entries.import.steps.file.parsing", { fileName })}
        </p>
      )}

      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">
          {t(`entries.import.errors.${error}`, {
            defaultValue: t("entries.import.errors.generic"),
          })}
        </p>
      )}

      <div>
        <button
          type="button"
          onClick={onBack}
          className="text-sm font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
        >
          {t("entries.import.back")}
        </button>
      </div>
    </div>
  );
}
