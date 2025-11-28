"""Wiki generation using Claude."""
import json
import logging
from typing import Any

import anthropic

from ..config import settings
from ..tools.neo4j_tools import neo4j_query

logger = logging.getLogger(__name__)

# Initialize Anthropic client
client = anthropic.Anthropic(api_key=settings.anthropic_api_key)


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


WIKI_PROMPT = """You are generating documentation for a code repository.

## Repository: {repo_name}

## Code Structure:
{code_structure}

## Instructions:
Generate a multi-page wiki with:

1. **Overview page** (slug: "overview", order: 1)
   - Project purpose based on the code
   - Architecture diagram in mermaid format
   - Key modules summary (one line each)

2. **Module pages** (one per directory, order: 2+)
   - Module purpose
   - Key functions with brief descriptions
   - How this module relates to others

## Output Format:
Return ONLY valid JSON (no markdown, no explanation) with this exact structure:
{{
  "pages": [
    {{
      "slug": "overview",
      "title": "Overview",
      "content": "# Project Name\\n\\nmarkdown content...",
      "order": 1,
      "parent_slug": null,
      "diagrams": [
        {{
          "id": "architecture",
          "title": "Architecture",
          "code": "graph TD\\n  A[Module] --> B[Module]"
        }}
      ]
    }},
    {{
      "slug": "module-name",
      "title": "Module Name",
      "content": "# Module Name\\n\\nmarkdown content...",
      "order": 2,
      "parent_slug": "overview",
      "diagrams": []
    }}
  ]
}}

IMPORTANT:
- Return ONLY the JSON, no other text
- Use \\n for newlines in content strings
- Keep content concise (200-400 words per page)
- Generate 3-6 pages total
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
        model="claude-sonnet-4-20250514",
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
        logger.error(f"Failed to parse Claude response: {e}")
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
