# his-review-sandbox

A small Go service shaped like the HIS repo, used to test the review workflow. Appointments belong to a facility; staff act only on their own facility's appointments.

## Commands
- `task test` — all tests. Never `go test ./...` directly; use the Taskfile.
- `task test -- -run TestName ./internal/scheduling/` — one test.
- Every test opens its own SQLite database in a temp dir (`internal/db.OpenTest`), runs all migrations and loads `internal/db/seed.sql`. No external database is needed.

## Rules
- Domain logic only through `internal/scheduling.Service`. Every Service method that reads or writes an appointment checks that the caller's facility owns it (`authorize`).
- HTTP: GET and POST only.
- Migrations: `internal/db/migrations/NNN_name.up.sql` + `.down.sql`, applied in order. Never edit a migration already on main. A migration must keep existing rows' data; the down must undo the up.
- No issue ids or review notes in source comments.
