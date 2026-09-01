# backend

Go HTTP API for family-finances.

## Commands (run from `backend/`)

```bash
go run .      # start the server (default port 8080, override with PORT)
go test ./... # run tests
go build .    # production build
```

## Endpoints

| Method | Path | Response |
| ------ | ---- | -------- |
| GET    | `/`  | `ok`     |
