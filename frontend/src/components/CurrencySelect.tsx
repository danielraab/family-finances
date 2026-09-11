import { useState } from "react";

/** Feature-detects Intl.supportedValuesOf, absent from older engines. */
export function listCurrencies(): string[] {
  const supportedValuesOf = (
    Intl as unknown as { supportedValuesOf?: (key: string) => string[] }
  ).supportedValuesOf;
  if (!supportedValuesOf) {
    return [];
  }
  try {
    return supportedValuesOf("currency").map((code) => code.toUpperCase());
  } catch {
    return [];
  }
}

/**
 * Currency `<select>` shared by the account form (create/edit) and the
 * profile settings' default-currency field. Options come from
 * `Intl.supportedValuesOf("currency")`; a `value` outside that list (an
 * already-saved currency this build's engine doesn't enumerate) is kept as
 * its own option rather than silently dropped.
 */
export function CurrencySelect({
  id,
  value,
  onChange,
  className,
  placeholder,
  required,
}: {
  id?: string;
  value: string;
  onChange: (value: string) => void;
  className?: string;
  placeholder?: string;
  required?: boolean;
}) {
  const [currencies] = useState(listCurrencies);

  return (
    <select
      id={id}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className={className}
      required={required}
    >
      {placeholder !== undefined && (
        <option value="" disabled>
          {placeholder}
        </option>
      )}
      {!currencies.includes(value) && value !== "" && (
        <option value={value}>{value}</option>
      )}
      {currencies.map((code) => (
        <option key={code} value={code}>
          {code}
        </option>
      ))}
    </select>
  );
}
