# syntax=docker/dockerfile:1
# erp-system：前端构建 + Go 编译 + 运行镜像

FROM node:22-alpine AS web
WORKDIR /src
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS server
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 app && apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=server /out/server ./server
COPY --from=web /src/dist ./web
RUN mkdir -p /app/uploads && chown app:app /app/uploads
USER app
EXPOSE 8889
# 不设置 ENTRYPOINT：镜像广场需要通过 command 覆盖启动命令，
# 由环境实例注入外部 MySQL 连接信息。
CMD ["./server"]
