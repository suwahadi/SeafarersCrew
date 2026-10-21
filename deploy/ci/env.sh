#!/bin/sh
set -eu

cp .env.example .env

set_var() {
  sed -i "s|^$1=.*|$1=$2|" .env
}

password=$(openssl rand -hex 16)
s3_key=$(openssl rand -hex 8)
s3_secret=$(openssl rand -hex 16)

set_var APP_ENV local
set_var APP_ENCRYPTION_KEY "$(openssl rand -base64 32)"
set_var REVALIDATE_SECRET "$(openssl rand -hex 32)"
set_var POSTGRES_PASSWORD "$password"
set_var DATABASE_URL "postgres://seafarers:${password}@postgres:5432/seafarers?sslmode=disable"
set_var LOCAL_S3_ACCESS_KEY "$s3_key"
set_var LOCAL_S3_SECRET_KEY "$s3_secret"
set_var S3_BOOTSTRAP_ACCESS_KEY_ID "$s3_key"
set_var S3_BOOTSTRAP_SECRET_ACCESS_KEY "$s3_secret"
