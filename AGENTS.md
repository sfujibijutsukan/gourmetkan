# AGENTS.md
# Guidance for coding agents in this repo

## Scope and reality check
- This repository currently only contains `docs/design.md`.
- There is no Go module, build script, or test harness in this repo yet.
- Commands and style guidance below are inferred from the design doc and
  typical Go + net/http + html/template + SQLite practices.
- Treat all commands as placeholders until actual code and tooling land.

## Source-of-truth docs
- Product/design spec: `docs/design.md`
- There are no Cursor or Copilot rules in this repo.

## Build, lint, test (expected/placeholder)
When code is added, prefer a simple Go-first toolchain.

### Build
- `go build ./...` (build all packages)
- `go build -o bin/app ./cmd/app` (if a cmd/app main exists)

### Run
- `go run ./cmd/app` (if a cmd/app main exists)
- If a config file/env is required, document it in README and in code.

### Lint / format
- `gofmt -w .` (format all Go files)
- `go vet ./...` (static checks)
- Optional: `golangci-lint run` if added later (do not assume by default)

### Test
- `go test ./...` (run all tests)

### Single test
- `go test ./path/to/pkg -run TestName` (exact test name)
- `go test ./path/to/pkg -run TestPrefix` (prefix match)
- `go test ./path/to/pkg -run 'TestName/Subtest'` (subtests)

### Database and data dirs
- SQLite file should live at `./data/app.db` per design doc.
- Ensure `./data` exists and is writeable.
- Backups expected in `./backup` with date stamp.

## Architecture expectations (from design doc)
- Single Go web server using `net/http`.
- HTML rendering via `html/template`.
- SQLite with `github.com/mattn/go-sqlite3` driver.
- OAuth via GitHub; CSRF protection on POST.
- Business logic: distance calculation (Haversine) and map URL parsing.

## Code style guidelines (Go)
Keep code simple and standard-library oriented. Use minimal dependencies.

### Formatting
- `gofmt` is mandatory; do not fight it.
- Keep lines reasonably short; avoid deeply nested logic.

### Imports
- Group imports in standard order: stdlib, third-party, local.
- Do not use dot imports.
- Prefer explicit package names; avoid renaming unless necessary.

Example:

```
import (
    "context"
    "database/sql"
    "net/http"

    "github.com/mattn/go-sqlite3"

    "example.com/app/internal/db"
)
```

### File layout and package structure
- Prefer `cmd/app` for main, `internal/...` for application code.
- Separate handlers/controllers from data access and pure logic.
- Keep templates in a dedicated `templates/` directory.

### Naming
- Exported identifiers use CamelCase with clear nouns/verbs.
- Unexported identifiers use camelCase.
- Use domain terms from `docs/design.md` (base, restaurant, review, session).
- Handler names should read like actions: `ListRestaurants`, `CreateReview`.

### Types and data models
- Use small structs for DB models and form/input structs.
- Prefer explicit types over `interface{}`.
- Use `sql.NullString` / `sql.NullInt64` for nullable DB fields.
- Keep JSON/HTML/template tags explicit and consistent when used.

### Error handling
- Always check errors; do not ignore.
- Wrap errors with context using `fmt.Errorf("...: %w", err)`.
- Return user-safe error messages; log internal detail server-side.
- HTTP handlers should map errors to status codes per design doc:
  - 400/422: validation errors
  - 401/403: auth errors
  - 404: not found
  - 500: generic server error

### Validation
- Validate all incoming fields on the server, even if UI validates.
- Follow ranges in design doc (rating 1-5, lat/long bounds, etc.).

### HTTP handlers
- Keep handlers thin; move logic to service functions.
- Use `http.MethodGet`/`http.MethodPost` constants.
- Keep response writes centralized to avoid double writes.

### Templates
- Use `html/template` (not `text/template`).
- Escape user input; no `template.HTML` unless trusted.
- Avoid complex logic inside templates; prepare data in handlers.

### Database access
- Use prepared statements for repeated queries.
- Use transactions for multi-step writes.
- Keep SQL in one place (e.g., `internal/db` package).
- Keep indices in schema migrations or startup checks.

### OAuth and sessions
- Validate OAuth `state` on callback.
- Store sessions server-side in SQLite as per design doc.
- CSRF token required for POST; reject missing or mismatched tokens.
- Cookies should be HttpOnly, SameSite=Lax, Secure in HTTPS.

### Logging
- Log request/response basics and error details.
- Avoid logging secrets (OAuth tokens, session IDs, CSRF tokens).

### Security
- Do not trust Google Maps URLs; parse and validate lat/lng bounds.
- Ensure access control checks for routes that require login.
- Use context timeouts for external HTTP calls (short URL expand).

### Concurrency
- Keep shared state minimal; use DB for state where possible.
- Avoid global mutable variables except config.

## Tests
- Place tests alongside packages (`*_test.go`).
- Prefer table-driven tests.
- Validate edge cases: invalid URLs, missing fields, auth failures.
- For distance calculations, include known coordinate fixtures.

## Documentation updates
- If you add build/test tools, update this file with exact commands.
- If you add Cursor/Copilot rules, include them here.

After implementing your changes, be sure to run a sanity check (go run or go test) and execute the full test suite with go test ./....

If any issues occur, carefully review the error messages and modify the code as necessary.
Then, run the sanity check and tests again to confirm that the problems have been resolved.

Once you reach a reasonable milestone and both implementation and tests are complete, run gofmt (and goimports if needed) to format the code.
Next, perform static analysis using golangci-lint (or go vet) to maintain code quality.

After that, commit your changes and proceed to the next task.