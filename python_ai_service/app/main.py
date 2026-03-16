from fastapi import FastAPI, HTTPException

from .schemas import (
    AnswerRequest,
    AnswerResponse,
    DocumentIn,
    DocumentOut,
    SearchRequest,
    SearchResult,
)
from .storage import index

app = FastAPI(title="AI Search Service", version="0.1.0")


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/stats")
def stats() -> dict[str, int]:
    return {"documents": index.count()}


@app.post("/documents", response_model=DocumentOut)
def add_document(doc: DocumentIn) -> DocumentOut:
    return index.add(doc)


@app.post("/documents/bulk", response_model=list[DocumentOut])
def add_documents(docs: list[DocumentIn]) -> list[DocumentOut]:
    return index.add_many(docs)


@app.get("/documents/{doc_id}", response_model=DocumentOut)
def get_document(doc_id: str) -> DocumentOut:
    try:
        return index.get(doc_id)
    except KeyError as exc:
        raise HTTPException(status_code=404, detail="document not found") from exc


@app.delete("/documents/{doc_id}")
def delete_document(doc_id: str) -> dict[str, str]:
    try:
        index.delete(doc_id)
    except KeyError as exc:
        raise HTTPException(status_code=404, detail="document not found") from exc
    return {"status": "deleted"}


@app.post("/search", response_model=list[SearchResult])
def search(req: SearchRequest) -> list[SearchResult]:
    return index.search(req.query, req.k)


@app.post("/answer", response_model=AnswerResponse)
def answer(req: AnswerRequest) -> AnswerResponse:
    results = index.search(req.query, req.k)
    if not results:
        return AnswerResponse(answer="No relevant content found.", sources=[])

    sentences: list[str] = []
    for result in results:
        for sentence in result.text.split("."):
            clean = sentence.strip()
            if clean:
                sentences.append(clean)

    answer_text = ". ".join(sentences[: req.max_sentences]).strip()
    if answer_text:
        answer_text += "."

    return AnswerResponse(answer=answer_text or "No relevant content found.", sources=results)
