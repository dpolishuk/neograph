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
    Query Neo4j for repository code structure with detailed function information.

    Args:
        repo_id: Repository ID

    Returns:
        Dictionary with files grouped by directory, including function details
    """
    query = """
    MATCH (repo:Repository {id: $repo_id})-[:CONTAINS]->(file:File)
    OPTIONAL MATCH (file)-[:DECLARES]->(fn:Function|Method)
    WITH file, collect({
        name: fn.name,
        signature: fn.signature,
        docstring: fn.docstring,
        startLine: fn.startLine,
        endLine: fn.endLine
    }) as functions
    OPTIONAL MATCH (file)-[:DECLARES]->(cls:Class)
    WITH file, functions, collect({
        name: cls.name,
        docstring: cls.docstring,
        startLine: cls.startLine,
        endLine: cls.endLine
    }) as classes
    RETURN {
        path: file.path,
        language: file.language,
        functions: functions,
        classes: classes
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

        # Filter out null entries from functions and classes
        if "functions" in file_data:
            file_data["functions"] = [f for f in file_data["functions"] if f.get("name")]
        if "classes" in file_data:
            file_data["classes"] = [c for c in file_data["classes"] if c.get("name")]

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


WIKI_PROMPT = """You are generating comprehensive, detailed documentation with diagrams for a code repository, similar to DeepWiki.

## Repository: {repo_name}

## Code Structure (includes functions, classes, docstrings):
{code_structure}

## Instructions:
Generate an extensive multi-page wiki with:
- Rich Mermaid diagrams throughout
- Detailed file-level documentation
- Complete function/method documentation with examples
- Edge cases and usage patterns

### Page Structure:

1. **Overview page** (slug: "overview", order: 1)
   - Project purpose and high-level description
   - **System Architecture diagram** (graph TD) showing all major components and their relationships
   - Technology stack and dependencies
   - Data flow overview
   - Key modules summary with descriptions
   - Getting started guide

2. **Module/Directory pages** (one per major directory, order: 2-10)
   - Module purpose and responsibilities
   - **Module Architecture diagram** showing internal components
   - List of files in the module with brief descriptions
   - How this module interacts with others
   - **Sequence diagram** for key operations

3. **File-level pages** (one per significant file, order: 11+)
   - File purpose and responsibilities
   - **Class diagram** if the file defines classes
   - Detailed function/method documentation:
     - Function signature
     - Purpose and behavior description
     - Parameters with types and descriptions
     - Return values
     - **Usage examples** (code snippets)
     - **Edge cases and error handling**
   - Internal data flow within the file
   - Dependencies and imports

### Documentation Depth for Functions:

For EACH function/method, provide:
1. **Signature**: The function declaration
2. **Description**: What it does in 2-3 sentences
3. **Parameters**: Each parameter with type and purpose
4. **Returns**: What the function returns
5. **Example Usage**: A realistic code example
6. **Edge Cases**: What happens with invalid input, empty data, etc.
7. **Related Functions**: Other functions it works with

Example function documentation format in content:
```
### `functionName(param1, param2)`

**Purpose**: Brief description of what this function does.

**Parameters**:
- `param1` (type): Description of first parameter
- `param2` (type): Description of second parameter

**Returns**: Description of return value

**Example**:
\\`\\`\\`python
result = functionName("value1", 42)
print(result)  # Expected output
\\`\\`\\`

**Edge Cases**:
- Empty input: Returns empty result
- Invalid type: Raises TypeError
```

### Diagram Types to Use:

1. **Flowcharts** (graph TD/LR) - Architecture, data flow, component relationships
   ```mermaid
   graph TD
       A[Component] --> B[Component]
       subgraph Module Name
           C[Internal] --> D[Internal]
       end
   ```

2. **Sequence Diagrams** - API calls, request handling, function call chains
   ```mermaid
   sequenceDiagram
       participant Caller
       participant Function
       participant Database
       Caller->>Function: call with params
       Function->>Database: query
       Database-->>Function: results
       Function-->>Caller: processed data
   ```

3. **Class Diagrams** - OOP structures, interfaces, data models
   ```mermaid
   classDiagram
       class ClassName {{
           +property: type
           +method(): returnType
       }}
       ClassName <|-- SubClass
   ```

4. **ER Diagrams** - Data models, database schemas, relationships
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
- Overview: system architecture + data flow (2-3 diagrams)
- Module pages: component diagram + sequence diagram (2 diagrams)
- File pages: class diagram + function flow (1-2 diagrams)

## Output Format:
Return ONLY valid JSON (no markdown wrapper, no explanation) with this structure:
{{
  "pages": [
    {{
      "slug": "overview",
      "title": "Project Overview",
      "content": "# {repo_name}\\n\\nComprehensive description...\\n\\n## System Architecture\\n\\n```mermaid\\ngraph TD\\n    A[Module] --> B[Module]\\n```\\n\\n## Technology Stack\\n\\n- Language: ...\\n- Framework: ...\\n\\n## Key Modules\\n\\n...",
      "order": 1,
      "parent_slug": null,
      "diagrams": [...]
    }},
    {{
      "slug": "module-name",
      "title": "Module Name",
      "content": "# Module Name\\n\\nDetailed purpose...\\n\\n## Files\\n\\n- file1.py: description\\n- file2.py: description\\n\\n## Architecture\\n\\n```mermaid\\ngraph TD\\n    A --> B\\n```",
      "order": 2,
      "parent_slug": "overview",
      "diagrams": [...]
    }},
    {{
      "slug": "module-name-filename",
      "title": "filename.py",
      "content": "# filename.py\\n\\nFile purpose...\\n\\n## Functions\\n\\n### `function1(param)`\\n\\n**Purpose**: ...\\n\\n**Parameters**:\\n- `param` (str): ...\\n\\n**Returns**: ...\\n\\n**Example**:\\n```python\\nresult = function1('test')\\n```\\n\\n**Edge Cases**:\\n- ...",
      "order": 11,
      "parent_slug": "module-name",
      "diagrams": [...]
    }}
  ]
}}

CRITICAL JSON FORMATTING RULES:
- Return ONLY valid JSON, no text before or after
- ALL newlines in strings MUST be escaped as \\n (two characters: backslash + n)
- NEVER use actual newline characters inside JSON string values
- Escape backticks in code examples: \\`\\`\\`language
- Example correct: "content": "Line 1\\nLine 2\\n\\`\\`\\`python\\ncode\\n\\`\\`\\`"
- Each page should have 1-3 diagrams minimum
- Generate 10-20 pages total for comprehensive coverage
- Include detailed function documentation with examples
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

    # Call Claude with extended token limit for comprehensive documentation
    response = client.messages.create(
        model="claude-sonnet-4-5-20250929",
        max_tokens=16384,
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
