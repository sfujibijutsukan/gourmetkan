# Gourmetkan

研究室向けのご飯屋共有アプリ（Go + SQLite）。

## Setup

```bash
export GITHUB_CLIENT_ID=your-client-id
export GITHUB_CLIENT_SECRET=your-client-secret
export BASE_URL=http://localhost:8080
export LISTEN_ADDR=:8080
export DATABASE_PATH=./data/app.db
```

```bash
go run ./cmd/app
```

## Docker Compose

```bash
export GITHUB_CLIENT_ID=your-client-id
export GITHUB_CLIENT_SECRET=your-client-secret
export BASE_URL=http://localhost:8080
docker compose up --build
```

Stop with:

```bash
docker compose down
```

## Notes

- SQLite DB: `./data/app.db`
- 初期拠点: OIC, BKC, 衣笠（固定順）
- `COOKIE_SECURE=true` で HTTPS 時の Secure Cookie を有効化
