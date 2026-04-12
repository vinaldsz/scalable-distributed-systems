# ChaosArena Album Store — Implementation Plan

## Context
Build a REST API for the ChaosArena `v1-album-store` contract (CS 6650), scored on:
- **Correctness**: 110 pts
- **p95 latency**: 80 pts

Greenfield Go service deployed on AWS EC2.

---

## Stack
| Component | Choice | Reason |
|---|---|---|
| Language/framework | Go 1.22 + Gin | Best p95 latency for load tests |
| Database | PostgreSQL 15 (local, same EC2) | Zero network latency for seq counter atomics |
| File storage | AWS S3 (public-read ACL) | Permanent public URL, ChaosArena fetches it |
| Async worker | Goroutine pool + buffered channel | No extra infra, natural Go pattern |
| DB driver | pgx/v5 + pgxpool | Fastest Go pg driver, direct SQL |
| S3 SDK | aws-sdk-go-v2 + s3/manager | Auto multipart for files >5 MB |

---

## Project Structure
```
album-store/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go
│   ├── db/
│   │   ├── db.go
│   │   └── migrations/001_init.sql
│   ├── models/models.go
│   ├── store/
│   │   ├── album_store.go
│   │   └── photo_store.go
│   ├── handlers/
│   │   ├── health.go
│   │   ├── albums.go
│   │   └── photos.go
│   ├── worker/pool.go
│   └── s3client/s3client.go
├── Dockerfile
├── .env.example
└── go.mod
```

---

## Checkpoints

---

### Checkpoint 1 — Scaffold + Config + DB + Migration
**Files:**
- `go.mod` / `go.sum`
- `internal/config/config.go`
- `internal/db/db.go`
- `internal/db/migrations/001_init.sql`
- `.env.example`
- `cmd/server/main.go` (stub)

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS albums (
    album_id    TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    owner       TEXT NOT NULL,
    photo_seq   BIGINT NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS photos (
    photo_id  TEXT PRIMARY KEY,
    album_id  TEXT NOT NULL REFERENCES albums(album_id),
    seq       BIGINT NOT NULL,
    status    TEXT NOT NULL DEFAULT 'processing',
    s3_key    TEXT,
    url       TEXT,
    UNIQUE (album_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_photos_album_id ON photos(album_id);
```

**Validation:**
```bash
cd chaosArena/album-store && go build ./...

psql $DATABASE_URL -f internal/db/migrations/001_init.sql
psql $DATABASE_URL -c "\dt"
# Expected: albums, photos

go run ./cmd/server
# Expected: "Server listening on :8080"
```

---

### Checkpoint 2 — Health + Album CRUD
**Files:**
- `internal/models/models.go`
- `internal/store/album_store.go`
- `internal/handlers/health.go`
- `internal/handlers/albums.go`
- Update `cmd/server/main.go` with routes

**Endpoints:**
- `GET /health`
- `PUT /albums/:album_id`
- `GET /albums/:album_id`
- `GET /albums`

**Validation:**
```bash
go run ./cmd/server &

curl -s http://localhost:8080/health
# → {"status":"ok"}

export AID=$(uuidgen | tr '[:upper:]' '[:lower:]')
curl -s -X PUT http://localhost:8080/albums/$AID \
  -H "Content-Type: application/json" \
  -d "{\"album_id\":\"$AID\",\"title\":\"Test\",\"description\":\"Desc\",\"owner\":\"a@b.com\"}"
# → {"album_id":"...","title":"Test","description":"Desc","owner":"a@b.com"}

curl -s http://localhost:8080/albums/$AID
# → same JSON

curl -s http://localhost:8080/albums
# → [{"album_id":"..."}]

curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/albums/does-not-exist
# → 404
```

---

### Checkpoint 3 — S3 Client + Worker Pool
**Files:**
- `internal/s3client/s3client.go`
- `internal/s3client/s3client_test.go`
- `internal/worker/pool.go`
- `internal/worker/pool_test.go`

**Validation:**
```bash
export S3_BUCKET=album-store-photos
export AWS_REGION=us-east-1

go test ./internal/s3client/... -v -run TestUploadAndDelete
# Expected: upload → 200 on public URL → delete → 403/404

go test ./internal/worker/... -v -run TestPool
# Expected: 100 jobs submitted, all processed, no deadlock on shutdown
```

---

### Checkpoint 4 — Photo Store + Photo Handlers
**Files:**
- `internal/store/photo_store.go`
- `internal/handlers/photos.go`
- Update `cmd/server/main.go` with photo routes

**Endpoints:**
- `POST /albums/:album_id/photos` → 202
- `GET /albums/:album_id/photos/:photo_id`
- `DELETE /albums/:album_id/photos/:photo_id` → 204

**Critical — atomic seq (single transaction):**
```sql
UPDATE albums SET photo_seq = photo_seq + 1 WHERE album_id = $1 RETURNING photo_seq;
INSERT INTO photos(photo_id, album_id, seq, status, s3_key) VALUES ($1,$2,$3,'processing',$4);
```

**Validation:**
```bash
go run ./cmd/server &

export PID=$(curl -s -X POST http://localhost:8080/albums/$AID/photos \
  -F "photo=@/path/to/test.jpg" | jq -r '.photo_id')
# → {"photo_id":"...","seq":1,"status":"processing"}

# Poll until completed
for i in {1..10}; do
  STATUS=$(curl -s http://localhost:8080/albums/$AID/photos/$PID | jq -r '.status')
  echo "Status: $STATUS"
  [ "$STATUS" = "completed" ] && break
  sleep 1
done
# → status="completed", url="https://..."

URL=$(curl -s http://localhost:8080/albums/$AID/photos/$PID | jq -r '.url')
curl -s -o /dev/null -w "%{http_code}" "$URL"
# → 200

curl -s -X DELETE http://localhost:8080/albums/$AID/photos/$PID
# → 204

curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/albums/$AID/photos/$PID
# → 404

curl -s -o /dev/null -w "%{http_code}" "$URL"
# → 403 or 404

# Concurrent seq uniqueness test
for i in {1..10}; do
  curl -s -X POST http://localhost:8080/albums/$AID/photos -F "photo=@/path/to/test.jpg" &
done
wait
psql $DATABASE_URL -c "SELECT seq FROM photos WHERE album_id='$AID' ORDER BY seq;"
# → seq values 1..10, no duplicates
```

---

### Checkpoint 5 — Wire Main + Graceful Shutdown + Dockerfile
**Files:**
- `cmd/server/main.go` (full wiring)
- `Dockerfile`

**HTTP Server config:**
```go
&http.Server{
    ReadTimeout:  10 * time.Minute,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
}
```

**Validation:**
```bash
go build -o bin/server ./cmd/server && echo "Build OK"

docker build -t album-store:local .
docker run --rm -p 8080:8080 --env-file .env album-store:local &

curl -s http://localhost:8080/health
# → {"status":"ok"}

kill -SIGTERM $(pgrep server)
# → logs "shutting down..." then exits cleanly
```

---

### Checkpoint 6 — EC2 Deploy + ChaosArena Submission
**Steps:**
1. Launch EC2 (t3.large or r6i.xlarge), Ubuntu 22.04, IAM role with S3 perms
2. Install Go 1.22 + PostgreSQL 15, create DB
3. Create S3 bucket (disable Block Public Access, ObjectOwnership=BucketOwnerPreferred)
4. Clone repo, `go build`, configure `.env`, run as systemd service
5. Open port 8080 in security group

**Validation:**
```bash
EC2=<your-ec2-ip>

curl http://$EC2:8080/health
# → {"status":"ok"}

# Submit to ChaosArena
curl -X POST https://chaosarena-alb-938452724.us-west-2.elb.amazonaws.com/submit \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"<your-email>\",\"nickname\":\"<nick>\",\"base_url\":\"http://$EC2:8080\",\"contract\":\"v1-album-store\"}"
# → ChaosArena returns correctness + p95 latency score
```

---

## Environment Variables
```
APP_PORT=8080
DATABASE_URL=postgres://albumuser:secret@localhost:5432/albumstore?sslmode=disable
DB_MAX_CONNS=40
DB_MIN_CONNS=5
AWS_REGION=us-east-1
S3_BUCKET=album-store-photos
WORKER_COUNT=20
WORKER_QUEUE_CAP=500
MAX_UPLOAD_MB=200
```

---

## Critical Files
| File | Purpose |
|---|---|
| `internal/store/photo_store.go` | Atomic seq tx — most load-sensitive |
| `internal/worker/pool.go` | Goroutine pool — affects p95 under load |
| `internal/s3client/s3client.go` | Public-read upload + delete |
| `cmd/server/main.go` | Graceful shutdown wiring |
| `internal/db/migrations/001_init.sql` | Schema with `photo_seq` on albums row |
