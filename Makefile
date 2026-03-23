.PHONY: build backend connection fanout frontend demo

backend:
	cd backend && go mod tidy
	cd backend && go build -o app ./cmd

connection:
	cd connection && go mod tidy
	cd connection && go build -o connection ./cmd/server

fanout:
	cd fanout && go mod tidy
	cd fanout && go build -o fanout ./cmd/worker

frontend:
	cd frontend && npm install && npm run build

build: backend connection fanout frontend

demo:
	$(MAKE) build
	docker compose up --build
