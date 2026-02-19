# Chords

A real-time chat application with a Go backend and React frontend. Users can register, log in, create and join chatrooms, and exchange messages over WebSockets.

## Project Focus

The current focus of this project is to improve WebSocket gateway functions (event routing, middleware pipeline, and message flow reliability).
The WebSocket gateway module is planned to be extracted into a separate component later.

## Tech Stack

| Layer      | Technologies |
|-----------|--------------|
| **API Service (`backend`)** | Go 1.24, [Gin](https://github.com/gin-gonic/gin), [GORM](https://gorm.io), JWT auth, [Redis](https://redis.io), [Logrus](https://github.com/sirupsen/logrus), Kafka client ([Sarama](https://github.com/IBM/sarama)) |
| **WS Gateway (`connection`)** | Go 1.24, [gorilla/websocket](https://github.com/gorilla/websocket), Kafka client ([Sarama](https://github.com/IBM/sarama)), protobuf/json event codecs |
| **Frontend** | React 19, React Router, Create React App |
| **Data** | SQLite (GORM), Redis (cache/sessions), Kafka (event bus) |
| **Deploy** | Docker, Docker Compose, Traefik (reverse proxy) |

## Project Structure

```
gochatroom/
├── backend/                 # Go API service (REST + business logic)
│   ├── cmd/main.go          # Entry point
│   ├── configs/config.yaml  # App/database config
│   ├── internal/
│   │   ├── app/             # DB & config
│   │   ├── cache/           # In-memory & Redis cache
│   │   ├── controller/      # HTTP handlers
│   │   ├── middleware/      # JWT auth, logger, load shedding
│   │   ├── model/           # GORM models
│   │   ├── repo/            # Data access
│   │   ├── routes/          # Route setup
│   │   ├── service/         # Business logic
│   │   └── ...
│   └── test/                # Integration tests
├── connection/              # WebSocket gateway service
│   ├── cmd/server/main.go   # Gateway entry point
│   ├── configs/config.yaml  # Gateway config (server/event/kafka)
│   ├── internal/
│   │   ├── gateway/         # Hub/connection transport logic
│   │   ├── handler/         # Event middleware pipeline abstractions
│   │   ├── event/codec/     # Event codec implementations
│   │   ├── platform/kafka/  # Kafka producer/consumer
│   │   └── service/         # Message service
│   └── proto/               # Gateway protobuf messages
├── frontend/                # React SPA
│   ├── src/
│   │   ├── components/      # Login, register, chatroom, messages, etc.
│   │   └── ...
│   └── package.json
├── docker-compose.yml       # App, frontend, Redis, Traefik
└── Dockerfile               # Backend image
```

## Prerequisites

- **Go** 1.24+ (for backend + connection)
- **Node.js** 18+ and npm (for frontend)
- **Redis** (required by backend)
- **Kafka** (required by backend/connection event flow)
- **Docker** & **Docker Compose** (optional, for full stack)

## Quick Start

### 1. Backend (local)

```bash
cd backend
go mod download
go run cmd/main.go
```

The API runs at **http://localhost:8080**. It expects:

- **Redis** at `redis:6379` (Docker network) or `localhost:6379` (local).  
  For local dev without Docker, start Redis (e.g. `redis-server`) and change `backend/internal/redisdb/redis.go` to use `Addr: "localhost:6379"` if needed.

- **SQLite** DB file `mydb.sqlite` in `backend/` (created automatically via config).

### 2. Connection Gateway (local)

```bash
cd connection
go mod download
go run cmd/server/main.go
```

The gateway runs at **http://localhost:8081** by default and serves WebSocket upgrades at **`/ws`**.
Kafka settings are in `connection/configs/config.yaml`.

### 3. Frontend (local)

```bash
cd frontend
npm install
npm start
```

The app runs at **http://localhost:3000** (Create React App default). Point it to the backend API via env (for example `REACT_APP_URL=http://localhost:8080/api`).

### 4. Full stack with Docker

```bash
docker compose up --build
```

- **Traefik** listens on host `:8000` and routes services by path.
- **Frontend**: `/`
- **API**: `/api`
- **WS Gateway**: `/ws`
- **Redis**: internal; exposed on `6379` for debugging if needed.
- **Kafka**: internal; exposed on `9092`.

## Configuration

### Backend

Edit `backend/configs/config.yaml`:

```yaml
app:
  port: 8080
frontend:
  port: 8081
database:
  dialect: sqlite
  dsn: mydb.sqlite
```

### Connection Gateway

Edit `connection/configs/config.yaml`:

```yaml
server:
  address: ":8081"
event:
  codec: "protobuf"
kafka:
  brokers:
    - "kafka:9092"
  consumer_group: "connection-ws-gateway"
  outbound_topics:
    - "notification"
  inbound_topic: "user-request"
```

### Redis

Backend connects to Redis in `backend/internal/redisdb/redis.go` (`Addr`, `Password`, `DB`). Use `redis:6379` when running in Docker, `localhost:6379` when running backend on the host.

## API Overview

All API routes are under `/api`. Auth uses JWT; send `Authorization: Bearer <token>` for protected routes.

| Area | Endpoints |
|------|-----------|
| **Auth** | `POST /api/auth/register`, `POST /api/auth/login`, `POST /api/auth/logout` |
| **Users** | `POST /api/users`, `GET /api/users` (auth) |
| **Chatrooms** | `POST/GET/DELETE /api/chatrooms`, `GET /api/chatrooms/:id`, `GET /api/chatrooms/search` (auth) |
| **Memberships** | `POST /api/memberships/add-user`, `GET /api/memberships/:username/chatrooms` (auth) |
| **Messages** | `POST /api/messages`, `GET /api/chatrooms/:id/messages`, `DELETE /api/messages/:id` (auth) |
| **WebSocket Gateway** | `GET /ws` (upgrade to WebSocket via the connection service) |

## Testing

**Backend**

```bash
cd backend
go test ./...
```

**Connection**

```bash
cd connection
go test ./...
```

**Frontend**

```bash
cd frontend
npm test
```
