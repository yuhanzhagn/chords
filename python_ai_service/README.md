# Python AI Microservice (Semantic Search + RAG-lite)

This independent service provides TF-IDF lexical search and an extractive "answer" endpoint. It does not call an LLM or embedding model and is not yet connected to the chat services.
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

## Development boundaries

See [agent instructions](AGENTS.md) and the repository [testing guide](../docs/testing.md).
The Dockerfile specifies Python 3.11. `make check-python` at the repository root
only checks source syntax; this service currently has no automated behavior tests.

The index uses one process-local lock and `data/store.json`. Multiple workers do
not share memory or a cross-process lock. Importing the app loads any existing
store. Use temporary data for storage tests and do not commit runtime documents.

For a first Docker build, the current Dockerfile requires a `data` directory:

```bash
mkdir -p python_ai_service/data
docker build -t chords-python-ai:dev python_ai_service
```

This is a development workaround for `COPY data ./data`; the image currently
copies any existing local data into itself. Build with an empty development data
directory. The service is not included in Compose, Kubernetes, or runtime CI.
