# Tasks

## 1. Configuration

- [x] 1.1 Add `Log{Level slog.Level, Requests RequestLogMode}` to `internal/config`, parsing `LOG_LEVEL` and `LOG_REQUESTS` per design D3; verify with `config_test.go` cases for defaults, every accepted value (including case and `warning`), and rejection of unknown values
- [x] 1.2 Document both variables and their interplay in `backend/.env.example` and `backend/AGENTS.md`; verify both files mention `LOG_LEVEL` and `LOG_REQUESTS`

## 2. Wiring

- [x] 2.1 Apply `slog.SetLogLoggerLevel(cfg.Log.Level)` in `main.go` right after config loads; verify with a live run (`LOG_LEVEL=warn` hides the startup `Info` lines)
- [x] 2.2 Make `withMiddleware` take the request-log mode (`all` / `errors` / `off`) and pass `cfg.Log.Requests` from `httpapi.New`; verify with middleware tests that capture log output for 200/404/429/500 in each mode, and that `off` keeps handler-written lines
