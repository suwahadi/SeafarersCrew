FROM node:22-bookworm-slim AS admin
WORKDIR /src
COPY admin/package.json admin/package-lock.json admin/.npmrc ./
RUN npm ci --no-audit --no-fund
COPY admin/ ./
RUN npm run build

FROM caddy:2-alpine
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY --from=admin /src/dist /srv/admin
