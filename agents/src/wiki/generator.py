"""Wiki generation using Claude."""
import json
import logging
import re
from typing import Any

import anthropic

from ..config import settings
from ..tools.neo4j_tools import neo4j_query

logger = logging.getLogger(__name__)


def get_anthropic_client() -> anthropic.Anthropic:
    """
    Create Anthropic client with flexible authentication.

    Supports:
    - ANTHROPIC_API_KEY: Standard API key authentication
    - ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL: OAuth/custom endpoint
    """
    kwargs = {}

    # Set base URL if provided (for custom endpoints)
    if settings.anthropic_base_url:
        kwargs["base_url"] = settings.anthropic_base_url

    # Prefer auth token if both base_url and auth_token are set (enterprise/custom)
    if settings.anthropic_base_url and settings.anthropic_auth_token:
        kwargs["api_key"] = settings.anthropic_auth_token
    elif settings.anthropic_api_key:
        kwargs["api_key"] = settings.anthropic_api_key

    return anthropic.Anthropic(**kwargs)


# Initialize Anthropic client
client = get_anthropic_client()


def get_code_structure(repo_id: str) -> dict[str, Any]:
    """
    Query Neo4j for repository code structure.

    Args:
        repo_id: Repository ID

    Returns:
        Dictionary with files grouped by directory
    """
    query = """
    MATCH (repo:Repository {id: $repo_id})-[:CONTAINS]->(file:File)
    OPTIONAL MATCH (file)-[:DECLARES]->(fn:Function)
    WITH file, collect({
        name: fn.name,
        signature: fn.signature,
        startLine: fn.startLine,
        endLine: fn.endLine
    }) as functions
    RETURN {
        path: file.path,
        language: file.language,
        functions: functions
    } as file
    ORDER BY file.path
    """

    # Validate repo_id is a valid UUID to prevent injection
    if not re.match(r'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$', repo_id, re.I):
        logger.error(f"Invalid repo_id format: {repo_id}")
        return {}

    results = neo4j_query(query.replace("$repo_id", f"'{repo_id}'"))

    # Group files by directory
    modules = {}
    for record in results:
        if "error" in record:
            continue
        file_data = record.get("file", record)
        path = file_data.get("path", "")

        # Extract directory as module name
        parts = path.split("/")
        if len(parts) > 1:
            module = parts[-2] if parts[-2] != "src" else parts[-1].replace(".py", "").replace(".go", "").replace(".ts", "")
        else:
            module = "root"

        if module not in modules:
            modules[module] = []
        modules[module].append(file_data)

    return modules


def fix_json_newlines(text: str) -> str:
    """
    Fix unescaped newlines inside JSON string values.

    Claude sometimes outputs actual newlines in JSON strings instead of \\n.
    This function attempts to fix that.
    """
    result = []
    in_string = False
    escape_next = False

    for char in text:
        if escape_next:
            result.append(char)
            escape_next = False
            continue

        if char == '\\':
            escape_next = True
            result.append(char)
            continue

        if char == '"' and not escape_next:
            in_string = not in_string
            result.append(char)
            continue

        if in_string and char == '\n':
            # Replace actual newline with escaped newline
            result.append('\\n')
        elif in_string and char == '\t':
            result.append('\\t')
        else:
            result.append(char)

    return ''.join(result)


WIKI_PROMPT = """You are generating comprehensive documentation with diagrams for a code repository, similar to DeepWiki.

## Repository: {repo_name}

## Code Structure:
{code_structure}

## Instructions:
Generate a multi-page wiki with rich Mermaid diagrams throughout.

### Page Structure:

1. **Overview page** (slug: "overview", order: 1)
   - Project purpose and description
   - **System Architecture diagram** (graph TD) showing all major components
   - Key modules summary with one-line descriptions
   - Include inline mermaid diagram in content using ```mermaid blocks

2. **Module pages** (one per major directory, order: 2+)
   - Module purpose and responsibilities
   - **Module Architecture diagram** showing internal components
   - Key functions/classes with descriptions
   - **Sequence diagram** if the module handles request flows
   - How this module relates to others

### Diagram Types to Use:

1. **Flowcharts** (graph TD/LR) - For architecture, data flow, component relationships
   ```mermaid
   graph TD
       A[Component] --> B[Component]
       subgraph Module Name
           C[Internal] --> D[Internal]
       end
   ```

2. **Sequence Diagrams** - For API calls, request handling, inter-module communication
   ```mermaid
   sequenceDiagram
       participant Client
       participant Server
       Client->>Server: Request
       Server-->>Client: Response
   ```

3. **Class Diagrams** - For OOP structures, interfaces, inheritance
   ```mermaid
   classDiagram
       class ClassName {{
           +property: type
           +method(): returnType
       }}
       ClassName <|-- SubClass
   ```

4. **ER Diagrams** - For data models, database schemas
   ```mermaid
   erDiagram
       ENTITY1 ||--o{{ ENTITY2 : relationship
       ENTITY1 {{
           string id
           string name
       }}
   ```

### Diagram Placement:
- Put diagrams in BOTH the "diagrams" array AND inline in "content" using ```mermaid blocks
- Each page should have at least 1-2 diagrams
- Overview should have system architecture + data flow diagrams
- Module pages should have component diagram + sequence diagram if applicable

## Output Format:
Return ONLY valid JSON (no markdown wrapper, no explanation) with this structure:
{{
  "pages": [
    {{
      "slug": "overview",
      "title": "Overview",
      "content": "# Project Name\\n\\nDescription...\\n\\n## System Architecture\\n\\n```mermaid\\ngraph TD\\n    A[Module] --> B[Module]\\n```\\n\\n## Key Modules\\n\\n...",
      "order": 1,
      "parent_slug": null,
      "diagrams": [
        {{
          "id": "system-architecture",
          "title": "System Architecture",
          "code": "graph TD\\n    subgraph Frontend\\n        UI[User Interface]\\n    end\\n    subgraph Backend\\n        API[API Server]\\n        DB[(Database)]\\n    end\\n    UI --> API\\n    API --> DB"
        }},
        {{
          "id": "data-flow",
          "title": "Data Flow",
          "code": "graph LR\\n    Input --> Process --> Output"
        }}
      ]
    }},
    {{
      "slug": "module-name",
      "title": "Module Name",
      "content": "# Module Name\\n\\nPurpose...\\n\\n## Architecture\\n\\n```mermaid\\ngraph TD\\n    A --> B\\n```\\n\\n## Key Functions\\n\\n...",
      "order": 2,
      "parent_slug": "overview",
      "diagrams": [
        {{
          "id": "module-architecture",
          "title": "Module Architecture",
          "code": "graph TD\\n    A[Handler] --> B[Service]\\n    B --> C[Repository]"
        }}
      ]
    }}
  ]
}}

CRITICAL JSON FORMATTING RULES:
- Return ONLY valid JSON, no text before or after
- ALL newlines in strings MUST be escaped as \\n (two characters: backslash + n)
- NEVER use actual newline characters inside JSON string values
- Example correct: "content": "Line 1\\nLine 2\\n```mermaid\\ngraph TD\\n    A --> B\\n```"
- Example WRONG: "content": "Line 1
Line 2" (this breaks JSON parsing)
- Each page should have 1-3 diagrams minimum
- Use subgraphs in flowcharts to group related components
- Generate 4-8 pages total with comprehensive diagrams
- Make diagrams detailed with meaningful node names from the actual code
"""


def generate_wiki(repo_id: str, repo_name: str) -> dict[str, Any]:
    """
    Generate wiki pages for a repository using Claude.

    Args:
        repo_id: Repository ID
        repo_name: Repository name for display

    Returns:
        Dictionary with 'pages' list
    """
    # Get code structure from Neo4j
    modules = get_code_structure(repo_id)

    if not modules:
        return {
            "pages": [{
                "slug": "overview",
                "title": "Overview",
                "content": f"# {repo_name}\n\nNo code structure found. Please ensure the repository has been indexed.",
                "order": 1,
                "parent_slug": None,
                "diagrams": []
            }]
        }

    # Format code structure for prompt
    code_structure = json.dumps(modules, indent=2)

    # Build prompt
    prompt = WIKI_PROMPT.format(
        repo_name=repo_name,
        code_structure=code_structure
    )

    logger.info(f"Generating wiki for {repo_name} with {len(modules)} modules")

    # Call Claude
    response = client.messages.create(
        model="claude-sonnet-4-5-20250929",
        max_tokens=8192,
        messages=[{"role": "user", "content": prompt}]
    )

    # Extract response text
    response_text = ""
    for block in response.content:
        if hasattr(block, "text"):
            response_text += block.text

    # Parse JSON response
    try:
        # Try to find JSON in response
        response_text = response_text.strip()
        if response_text.startswith("```"):
            # Remove markdown code blocks
            lines = response_text.split("\n")
            response_text = "\n".join(lines[1:-1])

        result = json.loads(response_text)
        logger.info(f"Generated {len(result.get('pages', []))} wiki pages")
        return result
    except json.JSONDecodeError as e:
        logger.warning(f"Initial JSON parse failed: {e}, attempting to fix newlines")
        # Try to fix unescaped newlines in JSON strings
        try:
            # Fix newlines inside JSON string values
            fixed_text = fix_json_newlines(response_text)
            result = json.loads(fixed_text)
            logger.info(f"Generated {len(result.get('pages', []))} wiki pages (after fixing)")
            return result
        except json.JSONDecodeError as e2:
            logger.error(f"Failed to parse Claude response even after fix: {e2}")
            logger.error(f"Response was: {response_text[:500]}")
            return {
                "pages": [{
                    "slug": "overview",
                    "title": "Overview",
                    "content": f"# {repo_name}\n\nWiki generation failed. Please try again.",
                    "order": 1,
                    "parent_slug": None,
                    "diagrams": []
                }]
            }
