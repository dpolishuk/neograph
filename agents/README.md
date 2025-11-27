# NeoGraph Agents Service

Python-based agent service providing Claude-powered code intelligence capabilities.

## Setup

### Install Dependencies

```bash
pip install -r requirements.txt
```

Or using the pyproject.toml:

```bash
pip install -e .
```

### Environment Variables

Create a `.env` file with:

```env
ANTHROPIC_API_KEY=your_api_key_here
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
```

## Running the Service

Start the FastAPI server:

```bash
uvicorn src.server:app --host 0.0.0.0 --port 8001 --reload
```

## API Endpoints

### POST /chat

Chat with Claude about your code.

**Request:**
```json
{
  "message": "Find authentication code",
  "repo_id": "optional-repo-id",
  "agent_type": "explorer"
}
```

**Response:**
```json
{
  "response": "Claude's response here",
  "tool_calls": []
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "ok"
}
```

## Architecture

- `src/server.py` - FastAPI application with CORS and Claude integration
- `src/config.py` - Environment configuration management
- Future: MCP tools for Neo4j integration
