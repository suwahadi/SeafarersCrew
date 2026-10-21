# Seafarers Crew

Single-admin platform for an offshore manning agency: crew records, STCW compliance, contracts, rotation, documents and a public website with a career portal.

- Product specification: [PRD.md](PRD.md)
- Engineering rules for humans and AI agents: [AGENTS.md](AGENTS.md)

## Repository layout

```
backend/   Go API, workers, CLI (cmd/seafarers)
admin/     React admin SPA (Vite + TypeScript)
web/       Next.js public website
deploy/    Caddyfile, Dockerfiles
docs/      restore procedure and architecture notes
```

## Run with Docker Compose

```sh
cp .env.example .env
# fill APP_ENCRYPTION_KEY (openssl rand -base64 32), REVALIDATE_SECRET (openssl rand -hex 32),
# POSTGRES_PASSWORD and the password inside DATABASE_URL
docker compose up -d --build
docker compose exec api /app/seafarers admin:create --name="Your Name" --email=you@example.com
```

`admin:create` prints a temporary password and ten one-time recovery codes. Store the codes offline.

Use `docker compose --profile s3 up -d` to run the bundled S3-compatible store (SeaweedFS) instead of an external S3 bucket. MinIO no longer publishes free container images, so SeaweedFS replaces it.

## CLI

| Command | Purpose |
| --- | --- |
| `seafarers serve` | Run the HTTP API |
| `seafarers migrate up\|down` | Apply all migrations or roll back one step |
| `seafarers db:seed` | Load reference data and the company and legal pages (idempotent) |
| `seafarers db:seed --demo [--force]` | Also load demo crew, contracts, vacancies, applicants and website content; refuses in production without `--force` |
| `seafarers admin:create --name= --email= [--password=]` | Create the administrator and print recovery codes |
| `seafarers admin:reset-password --email=` | Print a temporary password and revoke all sessions |
| `seafarers backup:run` | Dump the database to the private backup folder of the storage bucket now |
| `seafarers healthcheck` | Container health probe |

The stack serves the public website at `https://<DOMAIN>`, the admin panel at `https://admin.<DOMAIN>` and the API at `https://api.<DOMAIN>`. Set `APP_URL`, `ADMIN_URL` and `API_URL` in `.env` to those origins.

## Scheduled jobs and backups

The API runs its own scheduler (guarded by Redis locks, so several replicas are safe): daily database backup at the time set in Settings, hourly closing of vacancies past their closing date, quotation expiry, publishing of scheduled news, dashboard refresh every five minutes, a monthly purge of files of records deleted over 90 days ago and a daily purge of applications older than 24 months. Backups and how to restore them are described in [docs/restore.md](docs/restore.md).

## Public website

`web/` is a Next.js app that renders on the server from the public API. Admin changes reach it within seconds: the API bumps cache tags and calls `POST /api/revalidate` on the site (`WEB_URL`, secret `REVALIDATE_SECRET`). CI runs Lighthouse on Home, a CMS page, a news article, Career and Contact and fails when performance drops below 85.

```sh
cd web
cp .env.example .env.local   # API_URL, PUBLIC_API_URL, APP_URL, REVALIDATE_SECRET
npm ci && npm run dev
```

## Admin development

```sh
cd admin
npm ci
npm run dev        # Vite on :5173, proxies /api/admin to :8080
npm test && npm run lint && npm run typecheck
```

## Backend development

Requires Go 1.27 and Docker (repository tests start PostgreSQL 17 and Redis 7 with testcontainers).

```sh
cd backend
go test ./...
golangci-lint run ./...
```

Set `SKIP_CONTAINER_TESTS=1` to skip the repository tests when Docker is unavailable.
