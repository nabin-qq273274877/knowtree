# knowtree 多阶段构建：Vue 前端 → go:embed 打入 Go 二进制 → 精简运行层
# 最终镜像约 20-30MB

# ---- 阶段 1：前端 ----
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
RUN corepack enable
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm build

# ---- 阶段 2：Go 编译（内嵌前端产物）----
FROM golang:1.24-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /out/knowtree ./cmd/knowtree

# ---- 阶段 3：运行层 ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=backend /out/knowtree /usr/local/bin/knowtree

ENV KNOWTREE_DATA_DIR=/app/data \
    KNOWTREE_ADDR=0.0.0.0:3000
VOLUME /app/data
EXPOSE 3000

ENTRYPOINT ["knowtree"]
