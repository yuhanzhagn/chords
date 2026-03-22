# Chords

Chords is a real-time chat system built as a multi-service Go application with a React frontend.  
It supports user auth, chatroom membership, message persistence, and live event delivery.

## Project Focus

This project is a multi‑service chat system with three core runtime services:
- **`backend`**: REST APIs, auth, persistence, and business logic.
- **`connection`**: WebSocket gateway for real‑time clients.
- **`fanout`**: Kafka consumer that targets the right gateways via Redis.

The gateway (`connection`) is the real‑time execution layer. It:
- maintains client WebSocket sessions,
- runs an inbound middleware pipeline (auth, rate limit, validation, routing),
- forwards inbound chat events to Kafka,
- accepts fanout HTTP pushes and broadcasts to connected clients.

The fanout worker (`fanout`) consumes outbound Kafka events, resolves room membership and gateway ownership via Redis, then delivers events to the appropriate gateway instances over HTTP.

## Tech Stack

| Layer      | Technologies |
|-----------|--------------|
| **WS Gateway (`connection`)** | Go 1.24, [gorilla/websocket](https://github.com/gorilla/websocket), middleware pipeline, Kafka producer + HTTP fanout handler ([Sarama](https://github.com/IBM/sarama)), protobuf/json codecs |
| **API Service (`backend`)** | Go 1.24, [Gin](https://github.com/gin-gonic/gin), [GORM](https://gorm.io), JWT issuance/validation, [Redis](https://redis.io), [Logrus](https://github.com/sirupsen/logrus), Kafka client ([Sarama](https://github.com/IBM/sarama)) |
| **Fanout Workers (`fanout`)** | Go 1.24, Kafka consumer ([Sarama](https://github.com/IBM/sarama)), Redis registry, HTTP fanout to gateways |
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
├── fanout/                  # Fanout worker service
│   ├── cmd/worker/main.go   # Worker entry point
│   ├── configs/config.yaml  # Worker config (kafka/redis/fanout)
│   ├── internal/
│   │   ├── fanout/          # HTTP dispatch to gateways
│   │   ├── kafka/           # Kafka consumer
│   │   └── registry/        # Redis registry lookups
│   └── proto/               # Kafka protobuf messages
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
- **Go** 1.24+ (for fanout worker)
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
It also exposes a separate HTTP server for `/fanout` so fanout workers can push outbound events over HTTP instead of the gateway consuming Kafka outbound topics directly.

JWT behavior in dev:
- `backend` generates JWTs (login),
- `connection` validates JWTs on WebSocket requests,
- both use the same hardcoded key: `dev-shared-jwt-secret`.

### 3. Fanout Worker (local)

```bash
cd fanout
go mod download
go run cmd/worker/main.go
```

The worker consumes outbound Kafka topics and posts fanout payloads to the gateway's `/fanout` endpoint.
Redis key conventions used by the worker:
- `room:{room_id}:users` set of user IDs in the room
- `user:{user_id}:gateway` gateway address hosting that user

### 3. Frontend (local)

```bash
cd frontend
npm install
npm start
```

The app runs at **http://localhost:3000** (Create React App default). Point it to the backend API via env (for example `REACT_APP_URL=http://localhost:8080/api`).

### 4. Full stack with Docker

```bash
make demo
```

- **Traefik** listens on host `:8000` and routes services by path.
- **Frontend**: `/`
- **API**: `/api`
- **WS Gateway**: `/ws`
- **Fanout**: internal (posts to `/fanout` on gateway)
- **Redis**: internal; exposed on `6379` for debugging if needed.
- **Kafka**: internal; exposed on `9092`.

`make demo` builds the local Go binaries (including `fanout`) and then runs `docker compose up --build`.

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
fanout:
  address: ":8082"
event:
  codec: "protobuf"
kafka:
  brokers:
    - "kafka:9092"
  inbound_topic: "user-request"
```

### Fanout Worker

Edit `fanout/configs/config.yaml`:

```yaml
kafka:
  brokers:
    - "kafka:9092"
  consumer_group: "fanout-workers"
  topics:
    - "notification"

redis:
  addr: "redis:6379"
  password: ""
  db: 0
  room_users_prefix: "room:"
  room_users_suffix: ":users"
  user_gateway_prefix: "user:"
  user_gateway_suffix: ":gateway"

fanout:
  gateway_path: "/fanout"
  request_timeout: 3s
```

### Redis

Backend connects to Redis in `backend/internal/redisdb/redis.go` (`Addr`, `Password`, `DB`). Use `redis:6379` when running in Docker, `localhost:6379` when running backend on the host.

## API Overview

All API routes are under `/api`. Auth uses JWT; send `Authorization: Bearer <token>` for protected routes.
In the current dev setup, the JWT is issued by `backend` and validated by both `backend` and `connection` with the same hardcoded key (`dev-shared-jwt-secret`).

| Area | Endpoints |
|------|-----------|
| **Auth** | `POST /api/auth/register`, `POST /api/auth/login`, `POST /api/auth/logout` |
| **Users** | `POST /api/users`, `GET /api/users` (auth) |
| **Chatrooms** | `POST/GET/DELETE /api/chatrooms`, `GET /api/chatrooms/:id`, `GET /api/chatrooms/search` (auth) |
| **Memberships** | `POST /api/memberships/add-user`, `GET /api/memberships/:username/chatrooms` (auth) |
| **Messages** | `POST /api/messages`, `GET /api/chatrooms/:id/messages`, `DELETE /api/messages/:id` (auth) |
| **WebSocket Gateway** | `GET /ws` (upgrade to WebSocket via the connection service) |
| **Fanout Ingress** | `POST /fanout` (internal, used by fanout workers) |

## Connection Gateway Architecture

The `connection` service is the real-time execution layer of the system.

Inbound path (`client -> connection -> Kafka`):
1. Client sends a WebSocket event.
2. `connection` runs middleware chain (JWT auth, rate limiting, event filtering/routing).
3. Message events are encoded and published to Kafka inbound topic.

Outbound path (`Kafka -> fanout -> connection -> client`):
1. `fanout` consumes outbound Kafka topics.
2. It looks up room membership and gateway ownership in Redis.
3. `fanout` posts to `/fanout` on the owning gateway with a targeted user list.
4. Matching connected clients receive broadcast messages over existing WebSocket sessions.

## Fanout Registry Keys

The fanout worker expects Redis to keep two mappings:
- **Room membership**: `room:{room_id}:users` is a set of user IDs in the room.
- **User ownership**: `user:{user_id}:gateway` is a string with the gateway host:port.

These key prefixes/suffixes are configurable in `fanout/configs/config.yaml`.

This split lets `backend` remain focused on HTTP business APIs while `connection` scales independently for high-concurrency real-time traffic.

## Future Development Aims

The next stage focuses on production readiness of the `connection` gateway and platform operations:

1. Move deployment from Docker Compose to Kubernetes
- replace local Compose-first topology with k8s manifests/Helm-style deployment,
- separate stateless gateway replicas from stateful dependencies (Kafka/Redis/DB),
- standardize config/secrets/health probes for cluster-native operations.

2. Add fallback behavior for dependency failures
- define clear fallback paths when Kafka or Redis is unavailable,
- apply buffering/retry/circuit-breaking policies to avoid full request-path failure,
- keep connection service behavior predictable during partial outages.

3. Improve graceful degradation and graceful shutdown
- support controlled draining of WebSocket sessions during rollout/termination,
- stop accepting new upgrades before process exit,
- flush/close sinks and consumers safely to reduce message loss risk.

4. Make horizontal scaling easier in Kubernetes
- keep `connection` stateless per pod and externalize shared state/events,
- strengthen partition/group routing patterns for multi-replica fan-out,
- tune autoscaling signals (connection count, queue lag, CPU/memory) for real-time traffic.

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
