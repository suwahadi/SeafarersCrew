FROM golang:1.27-trixie AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seafarers ./cmd/seafarers

FROM debian:trixie-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libvips42t64 ghostscript postgresql-client \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --home-dir /app --shell /usr/sbin/nologin seafarers
WORKDIR /app
COPY --from=build /out/seafarers /app/seafarers
USER seafarers
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=3 CMD ["/app/seafarers", "healthcheck"]
ENTRYPOINT ["/app/seafarers"]
CMD ["serve"]
