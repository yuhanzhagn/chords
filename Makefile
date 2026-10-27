.DEFAULT_GOAL := help
GO ?= go
NPM ?= npm
PYTHON ?= python3

.PHONY: help install-frontend build build-go backend connection fanout frontend build-frontend demo test test-go test-backend test-connection test-fanout test-frontend check check-docs check-python proto

help:
	@printf '%s\n' \
	  'install-frontend  Install locked frontend dependencies (npm ci)' \
	  'test-go          Test all three independent Go modules' \
	  'test-frontend    Run Jest once without watch mode' \
	  'test             Run Go and frontend tests (install dependencies first)' \
	  'build-go         Compile all Go packages without writing demo binaries' \
	  'build-frontend   Build the React production bundle' \
	  'check-docs       Check local inline Markdown file links' \
	  'check-python     Parse Python sources without importing or writing data' \
	  'check            Docs, Python syntax, Go tests, Jest, frontend build' \
	  'proto            Regenerate Go and TypeScript protobuf files' \
	  'build            Build Linux demo binaries and install/build frontend' \
	  'demo             Build and start isolated Docker Compose demo'

install-frontend:
	cd frontend && $(NPM) ci --legacy-peer-deps

test-backend:
	cd backend && $(GO) test ./...

test-connection:
	cd connection && $(GO) test ./...

test-fanout:
	cd fanout && $(GO) test ./...

test-go: test-backend test-connection test-fanout

test-frontend:
	cd frontend && CI=true $(NPM) test -- --watchAll=false --runInBand

test: test-go test-frontend

build-go:
	cd backend && $(GO) build ./...
	cd connection && $(GO) build ./...
	cd fanout && $(GO) build ./...

build-frontend:
	cd frontend && $(NPM) run build

check-docs:
	$(PYTHON) scripts/check-docs.py

check-python:
	$(PYTHON) scripts/check-python.py

check: check-docs check-python test-go test-frontend build-frontend

proto:
	bash scripts/generate-proto.sh

# These binaries are mounted into the Linux demo containers.
backend:
	cd backend && GOOS=linux CGO_ENABLED=0 $(GO) build -o app ./cmd

connection:
	cd connection && GOOS=linux CGO_ENABLED=0 $(GO) build -o connection ./cmd/server

fanout:
	cd fanout && GOOS=linux CGO_ENABLED=0 $(GO) build -o fanout ./cmd/worker

frontend: install-frontend
	$(MAKE) build-frontend

build: backend connection fanout frontend

demo:
	$(MAKE) build
	docker compose up --build
