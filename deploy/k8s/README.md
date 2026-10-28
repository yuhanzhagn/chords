# Kubernetes (kind) - backend + gateway + redis + kafka

This setup runs `backend`, `connection`, `redis`, and single-node `kafka` in kind.

**Scope:** these manifests are an incomplete development topology. They do not
include fanout, frontend or the Python service, and the connection Service does
not expose its internal fanout port 8082. The steps below validate API/WS ingress,
not the complete real-time message round trip. Backend preserves Redis on startup and requires an externally supplied JWT signing key. There are no persistent volumes
for SQLite, Redis or Kafka in these manifests.

See the [deployment agent guide](../AGENTS.md), [architecture](../../docs/architecture.md)
and [known gaps](../../docs/status.md). Offline validation from the repo root:

```bash
kubectl kustomize deploy/k8s/base
```

## Prerequisites

- kind
- kubectl
- Docker
- An ingress controller in cluster (manifests assume `ingressClassName: nginx`)

## 1) Create kind cluster

```bash
kind create cluster --name gochatroom --config deploy/k8s/kind/cluster.yaml
```

## 2) Build images and load into kind

```bash
docker build -f Dockerfile.backend -t gochatroom/backend:kind .
docker build -f Dockerfile.connection -t gochatroom/connection:kind .

kind load docker-image --name gochatroom gochatroom/backend:kind
kind load docker-image --name gochatroom gochatroom/connection:kind
```

## 3) Install ingress-nginx (if not already installed)

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s
```

## 4) Apply manifests

Set `JWT_SECRET` locally to at least 32 random bytes and independently generate `FANOUT_TOKEN` with at least 32 visible ASCII bytes without spaces (for example, use `openssl rand -hex 32` separately for each). Do not commit or print the values. Create the namespace and both Secrets before starting backend/gateway. The gateway requires FANOUT_TOKEN even though this topology still has no worker; a future worker must receive the same value. Across untrusted networks, use trusted TLS: the shared token alone does not prevent interception or replay.

```bash
kubectl apply -f deploy/k8s/base/namespace.yaml
kubectl -n gochatroom create secret generic chords-auth --from-literal=JWT_SECRET="$JWT_SECRET"
kubectl -n gochatroom create secret generic chords-fanout --from-literal=FANOUT_TOKEN="$FANOUT_TOKEN"
kubectl apply -k deploy/k8s/base
```

## 5) Verify

```bash
kubectl -n gochatroom get pods,svc,ingress
kubectl -n gochatroom logs deploy/redis
kubectl -n gochatroom logs deploy/kafka
kubectl -n gochatroom logs deploy/backend
kubectl -n gochatroom logs deploy/connection
```

From host:

- API: `http://localhost:8000/api/...`
- WS: `ws://localhost:8000/ws`
