# Gourmetkan

グルメ館 -ご飯屋さん共有アプリ

## Setup
### Docker Compose

```bash
export GITHUB_CLIENT_ID=your-client-id
export GITHUB_CLIENT_SECRET=your-client-secret
export BASE_URL=http://localhost:8080
docker compose up --build
```

Uploaded images are persisted in a volume mounted at `./static/uploads`.
To keep images across redeploys, avoid removing volumes.

Stop with:

```bash
docker compose down
```

If you run `docker compose down -v`, uploaded images will also be deleted.

## Migration
1. Copy the following data from the old PC to the new PC
- SQLite DB(Restaurant name, other information...): `./data/app.db`
- Uploaded images: `./static/uploads`

2. Docker compose
