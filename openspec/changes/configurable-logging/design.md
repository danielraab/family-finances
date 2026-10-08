# Design

## Context

- `main.go` never calls `slog.SetDefault`, so output goes through the default
  logger (the `log` package bridge: `2026/10/08 12:00:00 INFO msg k=v`) at
  `Info`.
- `httpapi.logRequests` writes the access line at `Info` after every request.

## Decisions

### D1. The level goes through `slog.SetLogLoggerLevel`

`main.go` calls `slog.SetLogLoggerLevel(cfg.Log.Level)` right after
`config.Load()`. It changes the level of the default logger without replacing
its handler, so the format operators already parse stays byte-identical.

- *Alternative — `slog.SetDefault(slog.New(slog.NewTextHandler(…)))`:*
  rejected, because it changes the line format as a side effect. A
  `LOG_FORMAT` could add that later.
- Lines written before config loads (a config error itself) still log at the
  default `Info`.

### D2. A request-log mode, not a request-log level

`withMiddleware(h, mode)` gets `cfg.Log.Requests`:

- `all` logs every request, as today.
- `errors` logs only when the recorded status is ≥ 400.
- `off` skips the logging middleware entirely, so it costs nothing.

Access lines stay at `Info` in every mode. Raising `LOG_LEVEL` to `warn`
therefore also hides them: the level is the global floor, and `LOG_REQUESTS`
is the finer switch beneath it. This is documented next to both variables.

- *Alternative — logging 4xx at `Warn` and 5xx at `Error`:* rejected for
  now, because it changes the labels on today's default output. It is easy to
  add if the `errors` mode should survive `LOG_LEVEL=warn`.

### D3. Parsing

`LOG_LEVEL` accepts `debug|info|warn|warning|error`, case-insensitive.
`LOG_REQUESTS` accepts `all|errors|off`, case-insensitive. Empty means the
default, and anything else fails `config.Load()`.

## Risks / Trade-offs

- **[Operator sets `LOG_LEVEL=warn` and loses request lines unexpectedly]** →
  This is documented in `.env.example`. `LOG_REQUESTS` is the control meant
  for request lines.
- **[`off` hides abuse]** → Rate-limit refusals have their own `Info` line,
  which `LOG_REQUESTS=off` doesn't affect.
