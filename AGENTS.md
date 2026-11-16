# AGENTS.md

Rules for any AI agent working in this repository. Read [PROJECT.md](PROJECT.md) for the product brief.

## Never, without explicit permission

- **Never commit, push, amend, rebase, tag or touch git history.** The owner always commits and pushes. Leave changes in the working tree and say what changed.
- **Never leave a server or long-running process running when you finish.** You may start the app (`go run . serve`, the built binary) as often and as long as you need to validate and test, but stop it before you report back, so the owner never finds the port busy. Ask first before starting anything else long-running (`docker run`, `docker compose up`).

Pushing to `main` deploys to production through GitHub Actions, so the git rule is not negotiable.

## Running locally (no Docker needed)

```sh
go run . serve            # http://127.0.0.1:8090, dashboard on /_/ (data in ./pb_data, gitignored)
go run . devseed          # fills an empty local database with invented products
go build -o nevera . && ./nevera serve   # same, as a binary
```

Agents may run these while validating, and must stop them when done; see the rule above.

Validate against a throwaway copy, never the owner's `pb_data` or port: `--dir <tmp>` and `--http 127.0.0.1:8091`. A data directory with no superuser makes `serve` open PocketBase's installer in the owner's default browser (one tab per start), so create one first: `go run . superuser upsert throwaway@example.com <password> --dir <tmp>`. Nothing a test run does may be visible to the owner: no browser tabs, no windows.

## Working agreement

- UI copy is Spanish and lives in one place. Code, comments, commits, issues and docs are English.
- Public repository: no secrets, no personal data, no real product data in seeds.
- Ask before decisions that are hard to change later. One question at a time.
- Run `go vet ./...` and `go test ./...` before reporting work as done.
- Schema changes are new migration files in `migrations/`; never edit one that has been deployed.
- The mockup (`Main.dc.html`, `Frigo*`) is gitignored local reference, not production code.
