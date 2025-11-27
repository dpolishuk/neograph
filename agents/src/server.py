"""FastAPI server for NeoGraph agents."""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Optional
import anthropic
import os

from .config import settings


app = FastAPI(title="NeoGraph Agents")

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize Anthropic client
client = anthropic.Anthropic(api_key=settings.anthropic_api_key)


class ChatRequest(BaseModel):
    """Request model for chat endpoint."""

    message: str
    repo_id: Optional[str] = None
    agent_type: str = "explorer"


class ChatResponse(BaseModel):
    """Response model for chat endpoint."""

    response: str
    tool_calls: list = []


@app.post("/chat", response_model=ChatResponse)
async def chat(request: ChatRequest):
    """
    Handle chat requests with Claude API.

    Args:
        request: Chat request with message, optional repo_id, and agent_type

    Returns:
        ChatResponse with Claude's response and any tool calls
    """
    # Call Claude API
    response = client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=4096,
        messages=[{"role": "user", "content": request.message}],
    )

    # Extract text response
    response_text = ""
    if response.content:
        for block in response.content:
            if hasattr(block, "text"):
                response_text = block.text
                break

    return ChatResponse(
        response=response_text,
        tool_calls=[]
    )


@app.get("/health")
async def health():
    """Health check endpoint."""
    return {"status": "ok"}
