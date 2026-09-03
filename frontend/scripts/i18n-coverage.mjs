// Reports, per non-English locale under src/i18n/locales/, what percentage
// of en.json's translation keys it also has (by key presence, not value
// quality), plus any keys it has that en.json doesn't ("extra"/orphaned).
// en.json is the source of truth (see frontend/AGENTS.md). Run via
// `node scripts/i18n-coverage.mjs`. CI runs this as a non-blocking report
// (job summary + PR comment) — it never gates merging or other checks.

import { appendFileSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const localesDir = resolve(here, "../src/i18n/locales");

const MARKER = "<!-- i18n-coverage-report -->";

function collectKeys(obj, prefix = "", out = new Set()) {
  for (const [key, value] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === "object" && !Array.isArray(value)) {
      collectKeys(value, path, out);
    } else {
      out.add(path);
    }
  }
  return out;
}

function readJson(file) {
  return JSON.parse(readFileSync(file, "utf8"));
}

function displayName(code) {
  try {
    return new Intl.DisplayNames(["en"], { type: "language" }).of(code);
  } catch {
    return code;
  }
}

function detailsBlock(title, keys) {
  if (keys.length === 0) return "";
  const items = keys
    .slice()
    .sort()
    .map((k) => `- \`${k}\``)
    .join("\n");
  return `\n<details><summary>${title} (${keys.length})</summary>\n\n${items}\n\n</details>\n`;
}

const enFile = join(localesDir, "en.json");
const enKeys = collectKeys(readJson(enFile));

const localeFiles = readdirSync(localesDir)
  .filter((f) => f.endsWith(".json") && f !== "en.json")
  .sort();

if (localeFiles.length === 0) {
  console.log("No non-English locale files found under", localesDir);
  process.exit(0);
}

const results = localeFiles.map((file) => {
  const code = file.replace(/\.json$/, "");
  const localeKeys = collectKeys(readJson(join(localesDir, file)));
  const missing = [...enKeys].filter((k) => !localeKeys.has(k));
  const extra = [...localeKeys].filter((k) => !enKeys.has(k));
  const covered = enKeys.size - missing.length;
  const coverage = enKeys.size === 0 ? 100 : (covered / enKeys.size) * 100;
  return { code, name: displayName(code), covered, total: enKeys.size, coverage, missing, extra };
});

const rows = results
  .map(
    (r) =>
      `| ${r.name} (\`${r.code}\`) | ${r.coverage.toFixed(1)}% | ${r.covered}/${r.total} | ${r.extra.length} |`,
  )
  .join("\n");

const details = results
  .map(
    (r) =>
      detailsBlock(`${r.name} — missing keys`, r.missing) +
      detailsBlock(`${r.name} — extra keys (not in en.json)`, r.extra),
  )
  .join("");

const report =
  `${MARKER}\n` +
  "### Translation coverage (against `en.json`)\n\n" +
  "| Language | Coverage | Keys | Extra |\n" +
  "|---|---|---|---|\n" +
  `${rows}\n` +
  details;

console.log(report);

const summaryPath = process.env.GITHUB_STEP_SUMMARY;
if (summaryPath) {
  appendFileSync(summaryPath, `\n${report}\n`);
}

// Written for the CI job's follow-up PR-comment step to read, independent of
// its own working directory. Harmless when run locally (RUNNER_TEMP is
// unset there, so this lands in the OS temp dir and nothing reads it back).
const outFile = join(process.env.RUNNER_TEMP ?? tmpdir(), "i18n-coverage-report.md");
writeFileSync(outFile, report);

const shortfall = results.filter((r) => r.coverage < 100);
if (shortfall.length > 0) {
  console.error(
    `\nBelow 100% coverage: ${shortfall.map((r) => `${r.code} (${r.coverage.toFixed(1)}%)`).join(", ")}`,
  );
  process.exitCode = 1;
}
