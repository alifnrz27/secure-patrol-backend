# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.24-alpine AS build
WORKDIR /src

# Download dependencies first so they are cached between builds.
COPY go.mod go.sum ./
RUN go mod download

# garble obfuscates the license code (names and string literals) so the checks
# are hard to find and patch in the binary. Only the license packages are
# obfuscated: the models must keep their names, because GORM derives table and
# column names from them. OBFUSCATE=false gives a plain build (e.g. for debugging).
ARG OBFUSCATE=true
ARG GARBLE_VERSION=v0.14.2
# garble patches the Go linker with git, which the alpine image does not include.
RUN if [ "$OBFUSCATE" = "true" ]; then \
        apk add --no-cache git && go install mvdan.cc/garble@${GARBLE_VERSION} ; \
    fi

COPY . .
# Pure Go (pgx, pigo, excelize): no cgo needed, the binary is fully static.
ENV CGO_ENABLED=0 GOOS=linux \
    GOGARBLE=secure-patrol-backend/pkg/license,secure-patrol-backend/modules/license/...,secure-patrol-backend/middleware
RUN if [ "$OBFUSCATE" = "true" ]; then \
        garble -literals build -trimpath -ldflags="-s -w" -o /out/secure-patrol-backend . ; \
    else \
        go build -trimpath -ldflags="-s -w" -o /out/secure-patrol-backend . ; \
    fi

# ---- run ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/secure-patrol-backend /app/secure-patrol-backend

# logs/ and storage/ are written at runtime; mount volumes on them to keep data.
RUN mkdir -p /app/logs/debug /app/logs/access /app/storage && chown -R app:app /app
USER app

EXPOSE 3000

# The port comes from PORT in /app/.env (or the environment), default 3000.
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD sh -c 'PORT_IN_FILE=$(sed -nE "s/^[[:space:]]*PORT[[:space:]]*=[[:space:]]*\"?([0-9]+)\"?.*/\1/p" /app/.env 2>/dev/null | tail -1); \
               wget -qO- "http://127.0.0.1:${PORT:-${PORT_IN_FILE:-3000}}/health" >/dev/null || exit 1'

ENTRYPOINT ["/app/secure-patrol-backend"]
