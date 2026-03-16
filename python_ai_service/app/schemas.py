from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class DocumentIn(BaseModel):
    text: str = Field(..., min_length=1)
    metadata: Dict[str, Any] = Field(default_factory=dict)
    id: Optional[str] = None


class DocumentOut(BaseModel):
    id: str
    text: str
    metadata: Dict[str, Any]
    created_at: str


class SearchRequest(BaseModel):
    query: str = Field(..., min_length=1)
    k: int = Field(5, ge=1, le=50)


class AnswerRequest(BaseModel):
    query: str = Field(..., min_length=1)
    k: int = Field(5, ge=1, le=50)
    max_sentences: int = Field(3, ge=1, le=8)


class SearchResult(BaseModel):
    id: str
    score: float
    text: str
    metadata: Dict[str, Any]


class AnswerResponse(BaseModel):
    answer: str
    sources: List[SearchResult]
