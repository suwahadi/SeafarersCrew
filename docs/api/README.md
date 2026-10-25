# API reference

Complete reference of the HTTP API: admin API (`/api/admin/v1`), public API (`/api/public/v1`) and health probes.

| File | Purpose |
| --- | --- |
| `openapi.yaml` | OpenAPI 3.1 specification, the source of truth for this folder |
| `index.html` | Self-contained viewer built from the specification (open it in any browser; it loads the Redoc library from a CDN) |
| `build.py` | Rebuilds `index.html` after `openapi.yaml` changes: `python3 docs/api/build.py` (needs PyYAML) |

## Viewing

- Open `index.html` in a browser, or
- import `openapi.yaml` into any OpenAPI tool (Swagger UI, Scalar, Redoc, Postman, Insomnia) to browse it or generate API clients.

## Authentication in short

The admin API uses a session cookie (`sc_session`) plus a CSRF header (`X-CSRF-Token`, equal to the `sc_csrf` cookie) on every state-changing request. There are no API keys or bearer tokens. The public API needs no authentication. Details are in the specification overview.

## Security and hand-over

- The specification contains no credentials, secrets or real personal data. All examples are fictional.
- Hand over secrets (server, database, object storage, email, Turnstile, revalidation secret) through a secure channel, never by chat or plain email, and rotate every secret after hand-over. Use `.env.example` as the list of required variables.
- Create the first admin account with `seafarers admin:create`. It prints a temporary password and recovery codes; the admin must change the password at first login. Store the codes offline.
- Do not publish the admin part of this documentation on the open internet. Share it as a file, or host it behind authentication. For third parties, share only the operations tagged `Public site` and `Public career`.
- The admin API is designed for the bundled admin application (same site, cookie-based). It is not a public integration API.
- Known limits of version 1: a single admin account, no two-factor authentication and no audit log.

## Keeping it in sync

Update `openapi.yaml` in the same change that adds or alters a route, request or resource, then run `python3 docs/api/build.py`.
