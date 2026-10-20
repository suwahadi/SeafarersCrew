# AGENTS.md — Seafarers Crew

Rules for every developer and AI coding agent working in this repository. Read this file before writing code. The product specification lives in the PRD; when this file and the PRD disagree on engineering practice, this file wins.

## 1. Repository layout

```
seafarers-crew/
├── AGENTS.md
├── docker-compose.yml
├── .env.example
├── backend/      Go API, workers, CLI
├── admin/        React admin SPA (Vite + TypeScript)
├── web/          Next.js public website (TypeScript)
├── deploy/       Caddyfile, Dockerfiles
└── docs/         restore.md, architecture notes
```

## 2. Backend structure (MVC-style, idiomatic Go)

Layers follow Laravel's mental model while respecting Go conventions: `internal/` for private code, small interfaces defined by the consumer, explicit dependency injection, no global mutable state.

```
backend/
├── cmd/seafarers/main.go        entry point: serve, migrate, db:seed, admin:*, backup:run
├── internal/
│   ├── app/                     bootstrap and dependency wiring
│   ├── config/                  env loading and typed config
│   ├── routes/                  admin.go, public.go
│   ├── middleware/              auth, csrf, ratelimit, cors, requestid, recover
│   ├── handler/                 controllers
│   │   ├── admin/
│   │   └── public/
│   ├── request/                 input structs and validation
│   ├── resource/                response structs and mappers
│   ├── service/                 business logic, one file per domain
│   ├── repository/              data access (pgx / sqlc)
│   ├── model/                   domain structs
│   ├── job/                     workers, queue, scheduler
│   ├── platform/                postgres, redis, s3, brevo, gotenberg, turnstile, imaging, pdf
│   └── support/                 publicid, slug, money, datetime, errors
├── database/
│   ├── migrations/              golang-migrate SQL files
│   └── seeders/                 reference and demo seeders
└── testdata/                    fixture files for tests
```

| Laravel | Seafarers Crew (Go) |
| --- | --- |
| routes/api.php | internal/routes |
| app/Http/Controllers | internal/handler |
| app/Http/Requests | internal/request |
| app/Http/Resources | internal/resource |
| app/Services | internal/service |
| Repositories / Eloquent queries | internal/repository |
| app/Models | internal/model |
| app/Http/Middleware | internal/middleware |
| app/Jobs, Console/Kernel | internal/job |
| database/migrations, seeders | database/migrations, database/seeders |
| artisan | cmd/seafarers |

Request flow: route → middleware → handler → service → repository.

- Handlers parse the request, call one service method, and return a resource. No business logic.
- Services own business rules and transactions.
- Repositories run SQL only. No business rules.
- File names: `snake_case.go`, one domain per file (`crew_service.go`, `crew_repository.go`).

## 3. Frontend structure

```
admin/src/
├── app/          router, providers, layout
├── pages/        one folder per module
├── components/   ui, datagrid, forms, editor
├── api/          typed API client and endpoint functions
├── hooks/
├── lib/          formatters, validators
└── types/

web/
├── app/          App Router routes, layout, sitemap.ts, robots.ts, not-found.tsx
├── components/   blocks, layout, ui
├── lib/          api client, seo helpers, formatters
└── public/       flags, static icons
```

- Admin: React 19, Vite, TypeScript, TanStack Query, TanStack Table, React Hook Form + Zod, TipTap, Tailwind CSS, Recharts.
- Web: Next.js App Router, TypeScript, Tailwind CSS, `next/font`, `next/image`.

## 4. Comments

- No comments in production code: no explanatory notes, no TODO/FIXME tags, no commented-out code, no decorative section banners.
- Make code self-explanatory through clear names and small functions.
- Test files are the exception: comments are allowed where they clarify a test case.
- Where a linter requires a doc comment on an exported Go identifier, keep it to one short line.

## 5. Go rules

- Code must pass `gofmt` and `golangci-lint` with no warnings.
- `context.Context` is the first parameter of every function that does I/O.
- Wrap errors with context using `fmt.Errorf("...: %w", err)`; never discard errors.
- No package-level mutable state; inject dependencies through constructors.
- No `panic` in request paths.
- SQL is parameterized only; never build queries by string concatenation.
- Money uses decimal types and `NUMERIC(18,2)`; never `float64`.
- Times are stored in UTC; convert to the configured timezone only for display.
- Run independent queries for one response in parallel with `errgroup`.
- CPU-heavy work (images, PDF, conversion, import, export, backup) runs in bounded worker pools, never unbounded goroutines.
- Never log personal data (passport, bank details, full addresses).

## 6. TypeScript rules

- `strict` mode on; no `any`.
- ESLint must pass with no warnings.
- Server data in the admin goes through TanStack Query; no manual fetch-in-effect patterns.
- Public pages render their data on the server.

## 7. Performance and UX rules

- Public pages must score 85 or higher on Google PageSpeed Insights (mobile and desktop). Lighthouse CI enforces this.
- Never reduce, truncate, defer or skip data fetching to improve speed or scores. Fetch everything the page needs; gain speed through caching, parallel queries and smaller payloads.
- Do not use skeletons or placeholders that hide data the server could have rendered.
- Show a spinner only for genuinely slow actions: form submit, upload, PDF generation, export, or route transitions longer than 300 ms.
- Reserve space for images and embeds to avoid layout shift.
- Animations: CSS transitions only, 150–300 ms, ease-out, `transform`/`opacity`/`grid-template-rows`. No heavy animation libraries, parallax or continuous decorative motion. Respect `prefers-reduced-motion`.
- FAQ accordions animate height with `grid-template-rows: 0fr → 1fr` and use `button` + `aria-expanded`.

## 8. Security rules

- Every admin route and admin API endpoint requires a valid session.
- Uploaded files go through the shared upload pipeline only: size check, name check, extension whitelist, magic bytes, deep validation, transform, random rename, private S3 storage.
- Files are served only through short-lived presigned URLs.
- Secrets never enter the repository, logs, or client bundles. Use `.env` for bootstrap values and encrypted Settings for integration keys.
- Sanitize all rich text and builder HTML server-side before storing and rendering.

## 9. Testing

- Backend: Go `testing` + `testify`, table-driven tests. Repository tests use testcontainers-go with PostgreSQL.
- Frontend: Vitest + Testing Library for both `admin` and `web`.
- Every business rule in PRD Section 11 must have a unit test.
- Name tests after the behaviour they verify.
- Run all tests before every commit.

## 10. Git and commits

- Commit messages are free-form but must clearly describe what changed and why, in plain English.
  - Good: `Add relief plan conflict check for overlapping dates`
  - Bad: `update`, `fix stuff`
- Never add co-author trailers or attribution for Anthropic, Claude, or any AI tool. No `Co-Authored-By:` lines and no `Generated with` lines.
- One logical change per commit; do not mix unrelated formatting changes.
- CI must be green before merging.

## 11. Before you finish a task

- [ ] `gofmt`, `golangci-lint`, ESLint and TypeScript checks pass.
- [ ] Unit tests added or updated and passing.
- [ ] No comments in non-test code.
- [ ] No data fetching was removed or deferred for performance.
- [ ] Commit message is descriptive and has no AI attribution.
