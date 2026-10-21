# Restoring Seafarers Crew from a backup

The API dumps the PostgreSQL database every day with `pg_dump` (custom format, compressed), uploads the dump to the private `backups/` folder of the configured S3 bucket, and deletes dumps older than the retention period (Settings > Backup, default 30 days). Failures show on the dashboard.

A backup contains **only the database**. Uploaded files (crew documents, CMS images, generated PDFs) stay in the object storage bucket, which has to be protected on its own: enable versioning or replication on the bucket at your storage provider.

Two secrets are needed to make a restored system usable and are **not** inside the dump:

- `APP_ENCRYPTION_KEY`: integration keys in Settings (Brevo, S3 secret, Turnstile) are stored encrypted with it. Without the original key those values cannot be read and have to be entered again. Keep a copy of `.env` in a password manager.
- The S3 credentials, which are part of the same encrypted Settings. Set `S3_BOOTSTRAP_*` in `.env` on a brand-new database, or enter them in Settings after the restore.

## 1. Find the backup

In the admin panel open Settings > Backup to see the history, or list the bucket:

```sh
aws s3 ls s3://<bucket>/backups/ --endpoint-url <endpoint>
```

Files are named `seafarers-YYYYMMDD-HHMMSS.dump` (UTC). Pick the newest one marked `succeeded`.

## 2. Download it

```sh
aws s3 cp s3://<bucket>/backups/seafarers-20261021-020000.dump ./restore.dump --endpoint-url <endpoint>
```

Any S3 client works, for example `mc cp` or `curl --aws-sigv4`.

## 3. Prepare an empty database

Use PostgreSQL 17 (the dump is not readable by older servers). On a new host:

```sh
cp .env.example .env
docker compose up -d postgres redis
```

Make sure `.env` has the original `APP_ENCRYPTION_KEY`, `POSTGRES_*` and `DATABASE_URL` values. To restore into a database that already exists, create a fresh one instead of overwriting the live one:

```sh
docker compose exec postgres psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE seafarers_restore"
```

## 4. Restore

```sh
docker compose cp restore.dump postgres:/tmp/restore.dump
docker compose exec postgres pg_restore \
  -U "$POSTGRES_USER" --dbname=seafarers --no-owner --exit-on-error /tmp/restore.dump
```

`--exit-on-error` stops at the first problem, so a damaged dump never ends up half applied. If the target database name differs, change `--dbname` and update `DATABASE_URL`.

## 5. Start the application

```sh
docker compose up -d
```

The API entrypoint runs `migrate up`, which does nothing when the dump is current, or applies newer migrations when the dump comes from an older release. Then check:

```sh
curl -fsS https://api.<domain>/readyz
docker compose exec api /app/seafarers backup:run
```

`/readyz` reports the database, Redis and storage. The manual backup proves that the restored system can write to storage again.

## 6. Verify

- Log in to `admin.<domain>`. If the password is lost, run `docker compose exec api /app/seafarers admin:reset-password --email=<email>`.
- Open Settings and confirm the integrations are filled in (they are only readable with the original `APP_ENCRYPTION_KEY`). Run the connection tests for Email and Storage.
- Open a crew record with a document and use the file link to confirm the files are reachable in the bucket.
- Open the public website and a CMS page.

## Restore drill

Rehearse the procedure before launch and after every major release:

1. Run a backup: `docker compose exec api /app/seafarers backup:run`.
2. Restore it into a throw-away database, following steps 2 to 4 with `--dbname=seafarers_restore`.
3. Compare row counts of the main tables (`crews`, `contracts`, `applicants`, `pages`, `files`) with production.
4. Run `DATABASE_URL=<restored url> /app/seafarers migrate up` and make sure it reports no change.
5. Drop the throw-away database.

## Manual backups

Settings > Backup has **Run backup now**, and `seafarers backup:run` does the same from the command line. Before risky operations (large imports, migrations), run one first.
