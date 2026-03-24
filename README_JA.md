# Chords (日本語)

Chords は Go のマルチサービス構成と React フロントエンドで作られたリアルタイムチャットシステムです。  
ユーザー認証、チャットルーム参加、メッセージ永続化、リアルタイム配信に対応しています。

## プロジェクト概要

本プロジェクトは 3 つの実行サービスで構成されます。
- **`backend`**: REST API、認証、永続化、ビジネスロジック
- **`connection`**: リアルタイム WebSocket ゲートウェイ
- **`fanout`**: Kafka を購読し、Redis を使って対象ゲートウェイへ振り分け

`connection` はリアルタイム実行層であり、以下を担います。
- WebSocket セッションの維持
- 受信イベントのミドルウェア処理（認証・レート制限・検証・ルーティング）
- 受信イベントの Kafka 送信
- fanout からの HTTP push を受け取り配信

`fanout` は Kafka のアウトバウンドイベントを購読し、Redis の部屋・ユーザー情報を用いて適切なゲートウェイへ HTTP 送信します。  
`connection` は受信イベント時に Redis を更新し、fanout が配信先を特定できるようにします。

## 技術スタック

| レイヤー | 技術 |
|---|---|
| **WS ゲートウェイ (`connection`)** | Go 1.24, gorilla/websocket, ミドルウェアパイプライン, Kafka プロデューサ + HTTP fanout 受信, protobuf/json |
| **API サービス (`backend`)** | Go 1.24, Gin, GORM, JWT, Redis, Logrus, Kafka (Sarama) |
| **Fanout ワーカー (`fanout`)** | Go 1.24, Kafka (Sarama), Redis レジストリ, HTTP fanout |
| **フロントエンド** | React 19, React Router, CRA |
| **データ** | SQLite, Redis, Kafka |
| **デプロイ** | Docker, Docker Compose, Traefik |

## クイックスタート

## 前提条件

- **Go** 1.24+（backend/connection/fanout）
- **Node.js** 18+ と **npm**（frontend）
- **Redis** / **Kafka**（ローカル起動時に必要。Docker Compose を使う場合は不要）
- **Docker** & **Docker Compose**（フルスタック起動用・任意）

### 1. Backend（ローカル）

```bash
cd backend
go mod download
go run cmd/main.go
```

API は **http://localhost:8080** で稼働します。

### 2. Connection（ローカル）

```bash
cd connection
go mod download
go run cmd/server/main.go
```

WS は **http://localhost:8081/ws**、fanout は **http://localhost:8082/fanout** で受けます。

### 3. Fanout（ローカル）

```bash
cd fanout
go mod download
go run cmd/worker/main.go
```

### 4. Frontend（ローカル）

```bash
cd frontend
npm install
npm start
```

### 5. Docker で全体起動

```bash
docker compose up
```

1 コマンドでフルスタックが起動します。Go サービスはコンテナ内で `go run` により実行されるため、事前ビルドは不要です。

- **Traefik**: ホスト `:8000` で待ち受け、パスでルーティング
- **Frontend**: `/`
- **API**: `/api`
- **WS Gateway**: `/ws`
- **Fanout**: 内部（gateway の `/fanout` に POST）
- **Redis**: 内部（必要なら `6379` でデバッグ可能）
- **Kafka**: 内部（`9092` で接続）

Make を使いたい場合は `make up` でも同じ動作になります。

## 設定（要点）

### Connection 設定例

```yaml
server:
  address: ":8081"
fanout:
  address: ":8082"
  advertise_addr: "connection:8082"
event:
  codec: "protobuf"
kafka:
  brokers:
    - "kafka:9092"
  inbound_topic: "user-request"
redis:
  addr: "redis:6379"
  password: ""
  db: 0
  room_users_prefix: "room:"
  room_users_suffix: ":users"
  user_gateway_prefix: "user:"
  user_gateway_suffix: ":gateway"
```

### Fanout 設定例

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

## 主要フロー

**Inbound（client → connection → Kafka）**
1. クライアントが WebSocket で送信
2. `connection` がミドルウェア処理
3. Kafka の inbound topic に送信

**Outbound（Kafka → fanout → connection → client）**
1. `fanout` が Kafka の outbound を購読
2. Redis から room と gateway を解決
3. `connection` の `/fanout` に HTTP 送信
4. ルーム内クライアントへ配信
