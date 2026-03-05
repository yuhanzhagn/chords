# Kubernetes (kind) - phase 1 (backend + gateway)

This phase moves `backend` and `connection` to Kubernetes first.

## Prerequisites

- kind
- kubectl
- Docker
- An ingress controller in cluster (manifests assume `ingressClassName: nginx`)

Important: `backend` and `connection` still depend on `redis:6379` and `kafka:9092`.
In this first phase, those services must already exist in the same Kubernetes cluster/namespace,
or pods will fail at startup.

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

```bash
kubectl apply -k deploy/k8s/base
```

## 5) Verify

```bash
kubectl -n gochatroom get pods,svc,ingress
kubectl -n gochatroom logs deploy/backend
kubectl -n gochatroom logs deploy/connection
```

From host:

- API: `http://localhost:8000/api/...`
- WS: `ws://localhost:8000/ws`
