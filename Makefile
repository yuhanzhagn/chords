.PHONY: build backend connection fanout demo

backend:
	cd backend && go build -o app ./cmd

connection:
	cd connection && go build -o connection ./cmd/server

fanout:
	cd fanout && go build -o fanout ./cmd/worker

build: backend connection fanout

demo:
	$(MAKE) build
	docker compose up --build
