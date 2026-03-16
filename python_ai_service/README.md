# Python AI Microservice (Semantic Search + RAG-lite)

This service provides simple semantic search and an "answer" endpoint based on TF-IDF ranking.
It runs independently and stores documents in a local JSON file.

## Quick start

```bash
cd python_ai_service
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --host 0.0.0.0 --port 8010
```

## API

- `GET /health` -> status
- `GET /stats` -> document count
- `POST /documents` -> add one document
- `POST /documents/bulk` -> add many documents
- `GET /documents/{id}` -> fetch one
- `DELETE /documents/{id}` -> delete
- `POST /search` -> top-k search
- `POST /answer` -> simple extractive answer from top-k results

### Example

```bash
curl -X POST http://localhost:8010/documents \
  -H 'Content-Type: application/json' \
  -d '{"text": "We deploy our Go backend with Docker and Kubernetes.", "metadata": {"room": "dev"}}'

curl -X POST http://localhost:8010/search \
  -H 'Content-Type: application/json' \
  -d '{"query": "how do we deploy", "k": 3}'
```

## Notes

- Storage: `data/store.json` (local, simple, no external DB required).
- This is intentionally small and easy to extend to real embeddings later.
