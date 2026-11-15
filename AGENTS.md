# AGENTS.md

Rules for any AI agent working in this repository. Read [PROJECT.md](PROJECT.md) for the product brief.

## Never, without explicit permission

- **Never commit, push, amend, rebase, tag or touch git history.** The owner always commits and pushes. Leave changes in the working tree and say what changed.
- **Never start servers or long-running processes** (`go run . serve`, the built binary, `docker run`, `docker compose up`, dev servers). Ask first, and stop anything you started when done.

Pushing to `main` deploys to production through GitHub Actions, so these rules are not negotiable.

## Working agreement

- UI copy is Spanish and lives in one place. Code, comments, commits, issues and docs are English.
- Public repository: no secrets, no personal data, no real product data in seeds.
- Ask before decisions that are hard to change later. One question at a time.
- Run `go vet ./...` and `go test ./...` before reporting work as done.
- Schema changes are new migration files in `migrations/`; never edit one that has been deployed.
- The mockup (`Main.dc.html`, `Frigo*`) is gitignored local reference, not production code.
