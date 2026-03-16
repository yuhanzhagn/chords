from __future__ import annotations

from __future__ import annotations

from datetime import datetime, timezone
from json import dumps, loads
from threading import Lock
from typing import Dict, List
from uuid import uuid4

import numpy as np
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity

from .config import DATA_PATH
from .schemas import DocumentIn, DocumentOut, SearchResult


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


class InMemoryIndex:
    def __init__(self) -> None:
        self._data_path = DATA_PATH
        self._lock = Lock()
        self._docs: Dict[str, DocumentOut] = {}
        self._vectorizer = TfidfVectorizer(stop_words="english")
        self._matrix = np.empty((0, 0))

    def _rebuild_matrix(self) -> None:
        texts = [doc.text for doc in self._docs.values()]
        if not texts:
            self._matrix = np.empty((0, 0))
            return
        self._matrix = self._vectorizer.fit_transform(texts)

    def load(self) -> None:
        if not self._data_path.exists():
            return
        raw = loads(self._data_path.read_text())
        with self._lock:
            self._docs = {doc["id"]: DocumentOut(**doc) for doc in raw}
            self._rebuild_matrix()

    def save(self) -> None:
        self._data_path.parent.mkdir(parents=True, exist_ok=True)
        payload = [doc.model_dump() for doc in self._docs.values()]
        self._data_path.write_text(dumps(payload, indent=2))

    def add(self, doc: DocumentIn) -> DocumentOut:
        doc_id = doc.id or str(uuid4())
        new_doc = DocumentOut(
            id=doc_id,
            text=doc.text,
            metadata=doc.metadata,
            created_at=utc_now(),
        )
        with self._lock:
            self._docs[doc_id] = new_doc
            self._rebuild_matrix()
            self.save()
        return new_doc

    def add_many(self, docs: List[DocumentIn]) -> List[DocumentOut]:
        added: List[DocumentOut] = []
        with self._lock:
            for doc in docs:
                doc_id = doc.id or str(uuid4())
                new_doc = DocumentOut(
                    id=doc_id,
                    text=doc.text,
                    metadata=doc.metadata,
                    created_at=utc_now(),
                )
                self._docs[doc_id] = new_doc
                added.append(new_doc)
            self._rebuild_matrix()
            self.save()
        return added

    def get(self, doc_id: str) -> DocumentOut:
        with self._lock:
            doc = self._docs.get(doc_id)
        if not doc:
            raise KeyError(doc_id)
        return doc

    def delete(self, doc_id: str) -> None:
        with self._lock:
            if doc_id not in self._docs:
                raise KeyError(doc_id)
            self._docs.pop(doc_id)
            self._rebuild_matrix()
            self.save()

    def search(self, query: str, k: int) -> List[SearchResult]:
        with self._lock:
            if not self._docs:
                return []
            query_vec = self._vectorizer.transform([query])
            sims = cosine_similarity(query_vec, self._matrix).flatten()
            order = sims.argsort()[::-1][:k]
            docs = list(self._docs.values())

        results = []
        for idx in order:
            doc = docs[int(idx)]
            results.append(
                SearchResult(
                    id=doc.id,
                    score=float(sims[int(idx)]),
                    text=doc.text,
                    metadata=doc.metadata,
                )
            )
        return results

    def count(self) -> int:
        with self._lock:
            return len(self._docs)


index = InMemoryIndex()
index.load()
