# ビルドステージ
FROM golang:1.27-alpine AS build

WORKDIR /src
# 依存モジュールを先に取得してキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /rss-bot .

# 実行ステージ
FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=build /rss-bot /app/rss-bot
# config.json があれば同梱 (無い場合は実行時に -v ./config.json:/app/config.json でマウント)
COPY config*.json ./

ENTRYPOINT ["/app/rss-bot"]
CMD ["-config", "/app/config.json"]
