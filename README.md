# MailerCloud — Email Engagement Analytics

A high-throughput event ingestion service with a live campaign dashboard.

**Stack**: Go · MySQL 8 · Vue 3 + Vite · Docker Compose

---

## Quick Start

### Prerequisites
- [Docker](https://www.docker.com/get-started) & Docker Compose v2+

### 1. Start all services

```bash
docker compose up --build
```

This starts:
| Service   | URL                          | Description                     |
|-----------|------------------------------|---------------------------------|
| Frontend  | http://localhost:3000         | Vue 3 dashboard                 |
| Backend   | http://localhost:8080         | Go API server                   |
| MySQL     | localhost:3306               | Database (auto-initialized)     |

### 2. Use the dashboard

1. Open **http://localhost:3000**
2. Enter a Campaign ID (e.g. `camp-1`) or click a quick pick
3. Click **📡 Track Campaign**
4. Use the **⚡ Load Generator** section to fire a burst of events
5. Watch the stats update live every 5 seconds

### 3. Run the CLI load generator (optional)

```bash
docker compose --profile tools run --rm loadgen
```

Override defaults with environment variables:

```bash
docker compose --profile tools run --rm \
  -e CAMPAIGN_ID=camp-2 \
  -e TOTAL_EVENTS=50000 \
  -e CONCURRENCY=100 \
  loadgen
```

---

## Local Development Setup

If you prefer to run the components natively on your host machine instead of inside Docker Compose, follow the steps below.

### 1. Spin up Databases (MySQL & Redis)
You need MySQL and Redis running. You can choose to run just the databases in Docker (highly recommended to save configuration time) or natively.

#### Option A: Run Databases in Docker (Recommended)
This starts only the database dependencies in the background, mapping them to your localhost:
```bash
docker compose up -d mysql redis
```
- **MySQL** will be accessible on port `3307` (due to the `3307:3306` port mapping in `docker-compose.yml`).
- **Redis** will be accessible on port `6379`.

#### Option B: Run Databases Natively
If running MySQL and Redis natively on your machine:
- Redis must be running on `localhost:6379`.
- MySQL must be running on `localhost:3306` (or your configured port).
- Create a database named `mailercloud`.
- Initialize the database schema:
  ```bash
  mysql -h localhost -u mailercloud -p mailercloud < db/migrations/001_init.sql
  ```

---

### 2. Run the Go Backend

1. Navigate to the backend directory:
   ```bash
   cd backend
   ```
2. Run the backend service.
   - If using **Option A (Docker Databases)**, you must specify `DB_PORT=3307` since the Docker container maps port `3306` to `3307` on the host:
     - **Windows (PowerShell)**:
       ```powershell
       $env:DB_PORT="3307"; go run main.go
       ```
     - **Windows (CMD)**:
       ```cmd
       set DB_PORT=3307 && go run main.go
       ```
     - **Linux/macOS**:
       ```bash
       DB_PORT=3307 go run main.go
       ```
   - If using **Option B (Native Databases)**, you can run directly (uses default port `3306`):
     ```bash
     go run main.go
     ```
     *(Note: If your local MySQL setup has different credentials, set the `DB_USER` and `DB_PASSWORD` environment variables before running).*

The API server will listen on `http://localhost:8080`.

---

### 3. Run the Vue Frontend

1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Start the Vite dev server:
   ```bash
   npm run dev
   ```
   Open your browser to [http://localhost:3000](http://localhost:3000). The dev server is preconfigured to proxy `/events` and `/campaigns` requests to your local Go backend running at `http://localhost:8080`.

---

### 4. Run the CLI Load Generator (Optional)

1. Navigate to the loadgen directory:
   ```bash
   cd loadgen
   ```
2. Run the tool to generate load against the local backend:
   ```bash
   go run main.go
   ```
   Override default settings using environment variables:
   - **Windows (PowerShell)**:
     ```powershell
     $env:TOTAL_EVENTS="50000"; $env:CONCURRENCY="100"; go run main.go
     ```
   - **Windows (CMD)**:
     ```cmd
     set TOTAL_EVENTS=50000 && set CONCURRENCY=100 && go run main.go
     ```
   - **Linux/macOS**:
     ```bash
     TOTAL_EVENTS=50000 CONCURRENCY=100 go run main.go
     ```

---

## API Endpoints

### `POST /events`

Ingest a single engagement event.

```json
{
  "event_id": "abc-123",
  "campaign_id": "camp-1",
  "type": "opened",
  "timestamp": "2026-06-17T10:00:00Z"
}
```

- **type**: one of `sent`, `opened`, `clicked`, `bounced`
- Returns `202 Accepted` — event is enqueued for batch processing
- Idempotent: duplicate `event_id` values are silently ignored

### `GET /campaigns/{id}/stats`

Returns aggregated counts for a campaign.

```json
{
  "sent": 1200,
  "opened": 843,
  "clicked": 312,
  "bounced": 45
}
```

---

## Project Structure

```
MailerCloud/
├── docker-compose.yml
├── DESIGN.md              # Architecture & scaling decisions
├── README.md              # This file
├── backend/
│   ├── Dockerfile
│   ├── main.go            # Entry point + HTTP server
│   ├── handler/
│   │   ├── events.go      # POST /events
│   │   └── stats.go       # GET /campaigns/{id}/stats
│   ├── ingestion/
│   │   └── batcher.go     # Channel-based batch writer
│   ├── model/
│   │   └── event.go       # Event struct + validation
│   └── db/
│       └── mysql.go       # Connection pool setup
├── frontend/
│   ├── Dockerfile
│   ├── nginx.conf         # Reverse proxy to backend
│   ├── index.html
│   └── src/
│       ├── App.vue
│       ├── index.css       # Design system
│       ├── api/client.js   # API client with timeout
│       └── components/
│           ├── Dashboard.vue
│           ├── StatsCard.vue
│           └── LoadGenerator.vue
├── loadgen/
│   ├── Dockerfile
│   └── main.go            # CLI burst fire tool
└── db/
    └── migrations/
        └── 001_init.sql   # Schema initialization
```

---

## Shutting Down

```bash
# Stop services
docker compose down

# Stop and remove all data
docker compose down -v
```
