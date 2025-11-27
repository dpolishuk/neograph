"""Neo4j tools for Claude agents to query the code graph."""
from neo4j import GraphDatabase
from typing import Any, Dict, List, Optional
import logging

from ..config import settings

logger = logging.getLogger(__name__)

# Initialize Neo4j driver
driver = GraphDatabase.driver(
    settings.neo4j_uri,
    auth=(settings.neo4j_user, settings.neo4j_password)
)


def get_tools() -> List[Dict[str, Any]]:
    """
    Return Claude tool definitions for Neo4j operations.

    Returns:
        List of tool definitions compatible with Claude API
    """
    return [
        {
            "name": "neo4j_query",
            "description": "Execute a Cypher query against the Neo4j database to explore code structure",
            "input_schema": {
                "type": "object",
                "properties": {
                    "query": {
                        "type": "string",
                        "description": "Cypher query to execute"
                    },
                },
                "required": ["query"],
            },
        },
        {
            "name": "blast_radius",
            "description": "Find all functions that depend on or are affected by changes to a given function",
            "input_schema": {
                "type": "object",
                "properties": {
                    "function_name": {
                        "type": "string",
                        "description": "Name of the function to analyze"
                    },
                    "depth": {
                        "type": "integer",
                        "description": "How many levels of dependencies to traverse",
                        "default": 3
                    },
                },
                "required": ["function_name"],
            },
        },
        {
            "name": "find_function",
            "description": "Search for functions by name pattern",
            "input_schema": {
                "type": "object",
                "properties": {
                    "pattern": {
                        "type": "string",
                        "description": "Function name pattern (supports wildcards with *)"
                    },
                },
                "required": ["pattern"],
            },
        },
    ]


def execute_tool(name: str, args: Dict[str, Any]) -> Any:
    """
    Execute a tool by name with given arguments.

    Args:
        name: Tool name to execute
        args: Tool arguments

    Returns:
        Tool execution result

    Raises:
        ValueError: If tool name is unknown
    """
    if name == "neo4j_query":
        return neo4j_query(args["query"])
    elif name == "blast_radius":
        return blast_radius(
            args["function_name"],
            args.get("depth", 3)
        )
    elif name == "find_function":
        return find_function(args["pattern"])
    else:
        raise ValueError(f"Unknown tool: {name}")


def neo4j_query(query: str) -> List[Dict[str, Any]]:
    """
    Execute arbitrary Cypher query against Neo4j.

    Args:
        query: Cypher query to execute

    Returns:
        List of result records as dictionaries
    """
    try:
        with driver.session() as session:
            result = session.run(query)
            records = []
            for record in result:
                # Convert record to dictionary
                records.append(dict(record))
            return records
    except Exception as e:
        logger.error(f"Error executing Cypher query: {e}")
        return [{"error": str(e)}]


def blast_radius(function_name: str, depth: int = 3) -> Dict[str, Any]:
    """
    Find all functions that depend on the given function (blast radius analysis).

    This finds both direct and transitive dependencies up to the specified depth.

    Args:
        function_name: Name of the function to analyze
        depth: How many levels of CALLS relationships to traverse

    Returns:
        Dictionary with function info and its dependents
    """
    query = """
    // Find the target function
    MATCH (target:Function {name: $function_name})
    OPTIONAL MATCH (target)<-[:DECLARES]-(file:File)

    // Find all functions that call this function (directly or indirectly)
    OPTIONAL MATCH path = (caller:Function)-[:CALLS*1..%d]->(target)
    WHERE caller <> target

    WITH target, file,
         collect(DISTINCT {
             name: caller.name,
             signature: caller.signature,
             distance: length(path)
         }) as dependents

    // Count direct vs transitive
    WITH target, file, dependents,
         size([d IN dependents WHERE d.distance = 1]) as direct_count,
         size([d IN dependents WHERE d.distance > 1]) as transitive_count

    RETURN {
        function: target.name,
        signature: target.signature,
        file: file.path,
        total_dependents: size(dependents),
        direct_dependents: direct_count,
        transitive_dependents: transitive_count,
        dependents: dependents
    } as result
    """ % depth

    try:
        with driver.session() as session:
            result = session.run(query, {"function_name": function_name})
            record = result.single()
            if record:
                return record["result"]
            else:
                return {
                    "error": f"Function '{function_name}' not found",
                    "function": function_name
                }
    except Exception as e:
        logger.error(f"Error in blast_radius: {e}")
        return {"error": str(e), "function": function_name}


def find_function(pattern: str) -> List[Dict[str, Any]]:
    """
    Search for functions by name pattern.

    Supports wildcards with *.
    Example: "process*" finds all functions starting with "process"

    Args:
        pattern: Function name pattern with optional wildcards

    Returns:
        List of matching functions with their metadata
    """
    # Convert * wildcards to Neo4j regex pattern
    regex_pattern = pattern.replace("*", ".*")

    query = """
    MATCH (fn:Function)
    WHERE fn.name =~ $pattern
    OPTIONAL MATCH (fn)<-[:DECLARES]-(file:File)
    OPTIONAL MATCH (file)<-[:CONTAINS]-(repo:Repository)

    // Count incoming and outgoing calls
    OPTIONAL MATCH (fn)-[:CALLS]->(called)
    WITH fn, file, repo, count(DISTINCT called) as calls_count

    OPTIONAL MATCH (caller)-[:CALLS]->(fn)
    WITH fn, file, repo, calls_count, count(DISTINCT caller) as called_by_count

    RETURN {
        id: fn.id,
        name: fn.name,
        signature: fn.signature,
        file_path: file.path,
        repo_name: repo.name,
        start_line: fn.startLine,
        end_line: fn.endLine,
        calls: calls_count,
        called_by: called_by_count
    } as function
    ORDER BY fn.name
    LIMIT 50
    """

    try:
        with driver.session() as session:
            result = session.run(query, {"pattern": f"(?i){regex_pattern}"})
            functions = []
            for record in result:
                functions.append(record["function"])
            return functions
    except Exception as e:
        logger.error(f"Error in find_function: {e}")
        return [{"error": str(e)}]


def close_driver():
    """Close Neo4j driver connection."""
    driver.close()
