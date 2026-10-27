# syntax=docker/dockerfile:1

# ---------- Backend (Go) ----------
FROM golang:1.26-alpine AS api-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/api ./cmd/api

FROM alpine:3.22 AS api
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app
COPY --from=api-build /out/api /usr/local/bin/api
USER app
ENV APP_HOST=0.0.0.0 APP_PORT=8080 GIN_MODE=release
EXPOSE 8080
CMD ["api"]

# ---------- Frontend (Vite build, disajikan nginx) ----------
FROM node:22-alpine AS web-build
WORKDIR /app
RUN npm install -g pnpm@12.11.2
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend ./
RUN pnpm build

FROM nginx:1.27-alpine AS web
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=web-build /app/dist /usr/share/nginx/html
EXPOSE 80
