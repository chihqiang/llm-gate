# ===== Stage 1: Go backend =====
FROM golang:1.25-alpine AS go-builder
ENV GOPROXY=https://goproxy.cn,direct
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/gate main.go

# ===== Stage 2: Next.js frontend =====
FROM node:20-alpine AS node-builder
ENV COREPACK_NPM_REGISTRY=https://registry.npmmirror.com
ENV PNPM_REGISTRY=https://registry.npmmirror.com
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@10 --activate && pnpm install --frozen-lockfile --registry $PNPM_REGISTRY --ignore-scripts
COPY web/ .
RUN BUILD_OUTPUT=standalone pnpm build

# ===== Stage 3: Runtime =====
FROM node:20-alpine

# Install nginx + supervisord
RUN apk add --no-cache nginx supervisor

# Create non-root user for application processes
RUN addgroup -S app && adduser -S app -G app

# Go binary
COPY --from=go-builder /app/bin/gate /usr/local/bin/gate

# Config
COPY config.yaml /app/config.yaml

# Next.js standalone server
COPY --from=node-builder /app/web/.next/standalone /app/web
COPY --from=node-builder /app/web/.next/static /app/web/.next/static
COPY --from=node-builder /app/web/public /app/web/public

# Set ownership so app user can read/write
RUN chown -R app:app /app

# Nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Supervisor config
COPY supervisord.conf /etc/supervisord.conf

EXPOSE 80

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
