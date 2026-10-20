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
| `seafarers db:seed` | Load reference data (idempotent) |
| `seafarers admin:create --name= --email= [--password=]` | Create the administrator and print recovery codes |
| `seafarers admin:reset-password --email=` | Print a temporary password and revoke all sessions |
| `seafarers healthcheck` | Container health probe |

## Backend development

Requires Go 1.27 and Docker (repository tests start PostgreSQL 17 and Redis 7 with testcontainers).

```sh
cd backend
go test ./...
golangci-lint run ./...
```

Set `SKIP_CONTAINER_TESTS=1` to skip the repository tests when Docker is unavailable.
