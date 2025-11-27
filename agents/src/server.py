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


def get_system_prompt(agent_type: str, repo_id: Optional[str] = None) -> str:
    """
    Get system prompt for the specified agent type.

    Args:
        agent_type: Type of agent (explorer, analyzer, doc_writer)
        repo_id: Optional repository ID to scope the agent to

    Returns:
        System prompt string
    """
    base_prompts = {
        "explorer": """You are a code exploration agent with access to a Neo4j graph database.
Your job is to help users find and understand code in their repositories.

Available tools:
- neo4j_query: Execute Cypher queries to explore the code graph
- find_function: Search for functions by name pattern (supports wildcards)
- blast_radius: Find all functions that depend on a given function

When asked to find code:
1. Use find_function to search for functions by name
2. Use neo4j_query to explore relationships and structure
3. Provide clear explanations with file paths and line numbers""",

        "analyzer": """You are a code impact analysis agent.
Your job is to analyze dependencies and potential blast radius of changes.

Available tools:
- blast_radius: Find all functions that depend on a given function
- neo4j_query: Execute custom graph queries for deeper analysis
- find_function: Locate specific functions

When asked about dependencies:
1. Use blast_radius to find impacted code
2. Categorize by severity (direct vs transitive dependencies)
3. Suggest testing priorities based on impact""",

        "doc_writer": """You are a documentation generation agent.
Your job is to create clear, comprehensive documentation for code.

Available tools:
- neo4j_query: Fetch code structure and relationships
- find_function: Locate functions to document
- blast_radius: Understand function dependencies and usage

When asked to document:
1. Fetch the code and its relationships
2. Generate markdown documentation
3. Include examples and usage patterns from the graph"""
    }

    prompt = base_prompts.get(agent_type, base_prompts["explorer"])

    if repo_id:
        prompt += f"\n\nCurrently exploring repository: {repo_id}"

    return prompt


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


@app.get("/health")
async def health():
    """Health check endpoint."""
    return {"status": "ok"}
