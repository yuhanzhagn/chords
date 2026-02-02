# Chords

A real-time chat application with a Go backend and React frontend. Users can register, log in, create and join chatrooms, and exchange messages over WebSockets.

## Tech Stack

| Layer      | Technologies |
|-----------|--------------|
| **Backend** | Go 1.24, [Gin](https://github.com/gin-gonic/gin), [GORM](https://gorm.io), [gorilla/websocket](https://github.com/gorilla/websocket), JWT auth, [Redis](https://redis.io), [Logrus](https://github.com/sirupsen/logrus) |
| **Frontend** | React 19, React Router, Create React App |
| **Data** | SQLite (GORM), Redis (cache/sessions) |
| **Deploy** | Docker, Docker Compose, Traefik (reverse proxy) |

## Project Structure

```
gochatroom/
├── backend/                 # Go API server
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
│   │   ├── websocket/       # WebSocket hub & connections
│   │   └── ...
│   └── test/                # Integration tests
├── frontend/                # React SPA
│   ├── src/
│   │   ├── components/      # Login, register, chatroom, messages, etc.
│   │   └── ...
│   └── package.json
├── docker-compose.yml       # App, frontend, Redis, Traefik
└── Dockerfile               # Backend image
```

## Prerequisites

- **Go** 1.24+ (for backend)
- **Node.js** 18+ and npm (for frontend)
- **Redis** (for cache/sessions; required by backend)
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

### 2. Frontend (local)

```bash
cd frontend
npm install
npm start
```

The app runs at **http://localhost:3000** (Create React App default). Point it at the backend (e.g. `http://localhost:8080`) via your app’s API base URL / env (e.g. `REACT_APP_API_URL`).

### 3. Full stack with Docker

```bash
docker compose up --build
```

- **Traefik** listens on port 80; frontend and API are routed by path.
- **Frontend**: `/`
- **API**: `/api`
- **Redis**: internal; exposed on `6379` for debugging if needed.
- Set `REACT_APP_URL` (or equivalent) in `docker-compose.yml` for the frontend to match your host (e.g. `13.158.200.71:80/api` in the sample).

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
| **WebSocket** | `GET /api/ws/:userID` (upgrade to WebSocket for real-time messages) |

## Testing

**Backend**

```bash
cd backend
go test ./...
```

**Frontend**

```bash
cd frontend
npm test
```
