"""FastAPI server for NeoGraph agents."""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Optional, List, Dict, Any
import anthropic
import os
import logging

from .config import settings
from .tools import get_tools, execute_tool
from .agents import (
    get_explorer_prompt,
    get_analyzer_prompt,
    get_doc_writer_prompt,
)
from .wiki import generate_wiki

logger = logging.getLogger(__name__)

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
    tool_calls: List[Dict[str, Any]] = []


class WikiGenerateRequest(BaseModel):
    """Request model for wiki generation."""
    repo_id: str
    repo_name: str


class WikiPage(BaseModel):
    """Single wiki page."""
    slug: str
    title: str
    content: str
    order: int
    parent_slug: Optional[str] = None
    diagrams: List[Dict[str, Any]] = []


class WikiGenerateResponse(BaseModel):
    """Response model for wiki generation."""
    pages: List[WikiPage]


def get_system_prompt(agent_type: str, repo_id: Optional[str] = None) -> str:
    """
    Get system prompt for the specified agent type.

    Args:
        agent_type: Type of agent (explorer, analyzer, doc_writer)
        repo_id: Optional repository ID to scope the agent to

    Returns:
        System prompt string
    """
    # Map agent types to their respective prompt functions
    prompt_functions = {
        "explorer": get_explorer_prompt,
        "analyzer": get_analyzer_prompt,
        "doc_writer": get_doc_writer_prompt,
    }

    # Get the appropriate prompt function, default to explorer
    prompt_func = prompt_functions.get(agent_type, get_explorer_prompt)

    # Generate prompt with optional repo_id
    return prompt_func(repo_id=repo_id)


@app.post("/chat", response_model=ChatResponse)
async def chat(request: ChatRequest):
    """
    Handle chat requests with Claude API.

    Args:
        request: Chat request with message, optional repo_id, and agent_type

    Returns:
        ChatResponse with Claude's response and any tool calls
    """
    # Get tools and system prompt
    tools = get_tools()
    system_prompt = get_system_prompt(request.agent_type, request.repo_id)

    # Start conversation with user message
    messages = [{"role": "user", "content": request.message}]
    tool_calls_log = []

    # Agentic loop: handle tool use until we get a final response
    max_iterations = 10
    for iteration in range(max_iterations):
        # Call Claude API
        response = client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=4096,
            tools=tools,
            messages=messages,
            system=system_prompt,
        )

        # Check stop reason
        if response.stop_reason == "end_turn":
            # Extract final text response
            response_text = ""
            for block in response.content:
                if hasattr(block, "text"):
                    response_text += block.text

            return ChatResponse(
                response=response_text,
                tool_calls=tool_calls_log
            )

        elif response.stop_reason == "tool_use":
            # Add assistant's response to messages
            messages.append({"role": "assistant", "content": response.content})

            # Execute all tool calls
            tool_results = []
            for block in response.content:
                if block.type == "tool_use":
                    tool_name = block.name
                    tool_input = block.input
                    tool_use_id = block.id

                    logger.info(f"Executing tool: {tool_name} with input: {tool_input}")

                    try:
                        # Execute the tool
                        result = execute_tool(tool_name, tool_input)

                        # Log tool call
                        tool_calls_log.append({
                            "tool": tool_name,
                            "input": tool_input,
                            "result": result
                        })

                        # Add tool result to messages
                        tool_results.append({
                            "type": "tool_result",
                            "tool_use_id": tool_use_id,
                            "content": str(result)
                        })
                    except Exception as e:
                        logger.error(f"Error executing tool {tool_name}: {e}")
                        tool_results.append({
                            "type": "tool_result",
                            "tool_use_id": tool_use_id,
                            "content": f"Error: {str(e)}",
                            "is_error": True
                        })

            # Add tool results to messages for next iteration
            messages.append({"role": "user", "content": tool_results})

        else:
            # Unexpected stop reason
            logger.warning(f"Unexpected stop reason: {response.stop_reason}")
            break

    # If we hit max iterations, return what we have
    return ChatResponse(
        response="Maximum iterations reached. Please try rephrasing your question.",
        tool_calls=tool_calls_log
    )


@app.post("/wiki/generate", response_model=WikiGenerateResponse)
async def wiki_generate(request: WikiGenerateRequest):
    """
    Generate wiki pages for a repository.

    Args:
        request: Wiki generation request with repo_id and repo_name

    Returns:
        WikiGenerateResponse with generated pages
    """
    logger.info(f"Generating wiki for repo {request.repo_id} ({request.repo_name})")

    try:
        result = generate_wiki(request.repo_id, request.repo_name)
        return WikiGenerateResponse(pages=result.get("pages", []))
    except Exception as e:
        logger.error(f"Failed to generate wiki: {e}", exc_info=True)
        # Return fallback response instead of crashing
        return WikiGenerateResponse(pages=[WikiPage(
            slug="overview",
            title="Overview",
            content=f"# {request.repo_name}\n\nWiki generation encountered an error. Please try again.",
            order=1,
            parent_slug=None,
            diagrams=[]
        )])


@app.get("/health")
async def health():
    """Health check endpoint."""
    return {"status": "ok"}
