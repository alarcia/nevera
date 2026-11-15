# nevera

Self-hosted, mobile-first tracker for what is in the household fridge: where each thing is, when it expires, whether it is opened and how much is left. Go + [PocketBase](https://pocketbase.io) (SQLite), server-rendered. See [PROJECT.md](PROJECT.md) for the product brief.

## Run locally

```sh
go run . serve            # app on http://127.0.0.1:8090, dashboard on /_/
go test ./...
```

The first run prints a link to create the dashboard superuser. Schema edits made in the dashboard while running with `go run` are written to `migrations/` as Go files; commit them.

## Deploy

Pushing to `main` runs [deploy.yml](.github/workflows/deploy.yml) on GitHub-hosted runners: it builds the ARM64 image (tests included), pushes it to `ghcr.io/alarcia/nevera:latest`, then SSHes into the Pi through Cloudflare Access and runs `docker compose pull && docker compose up -d` in `~/nevera`.

Repository secrets: `SSH_HOST`, `SSH_USER`, `SSH_KEY`, `CF_ACCESS_CLIENT_ID`, `CF_ACCESS_CLIENT_SECRET`.

On the Pi, `~/nevera` holds a copy of [docker-compose.yml](docker-compose.yml) and a `.env` based on [.env.example](.env.example). Create `HOST_DATA_PATH` as the deploy user before the first deploy, or Docker creates it as root.

The container listens on `127.0.0.1` only. Access goes through cloudflared + Cloudflare Access; the app has no login of its own. The dashboard (`/_/`) has its own superuser login:

```sh
docker exec nevera /nevera superuser upsert you@example.com 'a-long-password' --dir=/pb_data
```

Back up `HOST_DATA_PATH`.
