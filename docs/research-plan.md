# Self-Hosted DeepWiki с Neo4j: архитектура Code Intelligence

Neo4j — отличный выбор для code intelligence. Графовая модель идеально подходит для представления зависимостей кода, а встроенный vector search (с 2023 года) позволяет объединить семантический поиск и граф траверсал в одном хранилище.

## Почему Neo4j лучше PostgreSQL для code intelligence

| Критерий | Neo4j | PostgreSQL + pgvector |
|----------|-------|----------------------|
| **Граф зависимостей** | Нативные связи, O(1) traversal | JOIN-ы, деградация на глубине |
| **"Blast radius" анализ** | `MATCH path = (m)-[:CALLS*1..5]->(n)` | Recursive CTE, медленно |
| **Impact analysis** | Естественные Cypher запросы | Сложные SQL конструкции |
| **Vector search** | Встроенный HNSW index | pgvector extension |
| **Визуализация** | Neo4j Browser/Bloom из коробки | Требует отдельный инструмент |
| **Схема данных** | Гибкая, легко расширять | Миграции при изменениях |

**Ключевое преимущество:** Код — это граф по своей природе. AST, call graphs, dependency trees, import chains — всё это графовые структуры. Neo4j позволяет запрашивать их естественным образом.

---

## Нужны ли embeddings? — Ответ

**Да, embeddings нужны, и Neo4j делает их интеграцию элегантной.**

### Когда использовать vector search:

| Use Case | Vector Search | Graph Traversal | Hybrid |
|----------|:-------------:|:---------------:|:------:|
| "Найди код аутентификации" | ✅ | ❌ | ✅ |
| "Что вызывает эту функцию" | ❌ | ✅ | ❌ |
| "Похожий код + его зависимости" | ❌ | ❌ | ✅ |
| "Где используется DATABASE_URL" | ❌ | ✅ | ❌ |
| "Код для работы с JWT токенами" | ✅ | ❌ | ✅ |
| "Impact analysis изменения" | ❌ | ✅ | ❌ |

### Важный инсайт от Greptile:

Сходство между запросом и **raw code** = 0.7280
Сходство между запросом и **NL-описанием кода** = 0.8152 (+12%)

**Рекомендация:** Генерировать NL-описания для code chunks через LLM, затем embeddить описания.

---

## Neo4j Vector Search: embeddings + граф в одной БД

С Neo4j 5.11+ (2023) и улучшениями в 2025.10 доступен полноценный vector search на основе HNSW индекса:

```cypher
-- Создание vector index
CREATE VECTOR INDEX `code-embeddings` 
FOR (n:Function) ON (n.embedding)
OPTIONS {
  indexConfig: {
    `vector.dimensions`: 1024,
    `vector.similarity_function`: 'cosine'
  }
}

-- Гибридный запрос: vector search + graph traversal
CALL db.index.vector.queryNodes('code-embeddings', 10, $queryEmbedding)
YIELD node AS similar, score
WHERE score > 0.7

-- Найти все функции, которые вызывают похожий код
MATCH (caller:Function)-[:CALLS]->(similar)
RETURN caller.name, similar.name, score
ORDER BY score DESC
```

**Уникальная возможность:** Объединить семантическое сходство с графовыми связями в одном запросе.

---

## Архитектурная схема с Neo4j

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                    SELF-HOSTED DEEPWIKI + NEO4J                               │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                     GIT INTEGRATION LAYER                               │ │
│  │  ┌───────────┐  ┌───────────┐  ┌────────────┐  ┌─────────────────────┐ │ │
│  │  │  GitHub   │  │  GitLab   │  │  Bitbucket │  │  Webhooks           │ │ │
│  │  │  App      │  │  API      │  │  API       │  │  (push → reindex)   │ │ │
│  │  └─────┬─────┘  └─────┬─────┘  └──────┬─────┘  └──────────┬──────────┘ │ │
│  └────────┼──────────────┼───────────────┼───────────────────┼────────────┘ │
│           └──────────────┴───────────────┴───────────────────┘              │
│                                    │                                         │
│  ┌─────────────────────────────────▼───────────────────────────────────────┐ │
│  │                      INDEXING PIPELINE (Go/Kotlin)                      │ │
│  │                                                                         │ │
│  │  ┌──────────────────┐    ┌──────────────────┐    ┌────────────────────┐│ │
│  │  │   Tree-sitter    │    │   AST → Graph    │    │   NL Description   ││ │
│  │  │   Multi-lang     │───▶│   Transformer    │───▶│   Generator (LLM)  ││ │
│  │  │   Parser         │    │                  │    │                    ││ │
│  │  │                  │    │  • Functions     │    │ "This function     ││ │
│  │  │  Go, Kotlin, TS  │    │  • Classes       │    │  validates JWT     ││ │
│  │  │  Python, Java    │    │  • Imports       │    │  tokens..."        ││ │
│  │  └──────────────────┘    │  • Calls         │    └─────────┬──────────┘│ │
│  │                          │  • References    │              │           │ │
│  │                          └────────┬─────────┘              │           │ │
│  │                                   │                        │           │ │
│  │  ┌────────────────────────────────▼────────────────────────▼──────────┐│ │
│  │  │                    Embedding Generator                             ││ │
│  │  │  voyage-code-3 API или Qodo-Embed-1 (self-hosted)                  ││ │
│  │  │  • Embed NL descriptions (не raw code!)                            ││ │
│  │  │  • 1024 dimensions, cosine similarity                              ││ │
│  │  └────────────────────────────────┬───────────────────────────────────┘│ │
│  └───────────────────────────────────┼────────────────────────────────────┘ │
│                                      │                                       │
│  ┌───────────────────────────────────▼───────────────────────────────────┐  │
│  │                                                                        │  │
│  │                         NEO4J GRAPH DATABASE                           │  │
│  │                                                                        │  │
│  │   ┌─────────────────────────────────────────────────────────────────┐ │  │
│  │   │                     CODE PROPERTY GRAPH                          │ │  │
│  │   │                                                                  │ │  │
│  │   │    (Repository)──[:CONTAINS]──▶(File)                           │ │  │
│  │   │         │                         │                              │ │  │
│  │   │         │                    [:DECLARES]                         │ │  │
│  │   │    [:HAS_CONFIG]                  ▼                              │ │  │
│  │   │         │              ┌────────────────────┐                    │ │  │
│  │   │         ▼              │ (Function/Class)   │                    │ │  │
│  │   │    (EnvVar)            │  • name            │                    │ │  │
│  │   │      │                 │  • signature       │                    │ │  │
│  │   │      │                 │  • docstring       │                    │ │  │
│  │   │  [:USED_IN]            │  • embedding ◀─────┼── Vector Index     │ │  │
│  │   │      │                 │  • startLine       │   (HNSW)           │ │  │
│  │   │      ▼                 │  • endLine         │                    │ │  │
│  │   │  (ConfigFile)          └─────────┬──────────┘                    │ │  │
│  │   │      │                           │                               │ │  │
│  │   │  [:SETS_VALUE]          [:CALLS] │ [:IMPORTS] │ [:EXTENDS]       │ │  │
│  │   │      │                           ▼            ▼                  │ │  │
│  │   │      ▼                    (Function)    (Module)                 │ │  │
│  │   │   (UIConfig)                                                     │ │  │
│  │   │                                                                  │ │  │
│  │   └─────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                        │  │
│  │   ┌─────────────────────┐  ┌─────────────────────┐                    │  │
│  │   │   Vector Index      │  │   Full-text Index   │                    │  │
│  │   │   (HNSW, cosine)    │  │   (Lucene-based)    │                    │  │
│  │   │   code-embeddings   │  │   code-fulltext     │                    │  │
│  │   └─────────────────────┘  └─────────────────────┘                    │  │
│  │                                                                        │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                       │
│  ┌───────────────────────────────────▼───────────────────────────────────┐  │
│  │                      RETRIEVAL LAYER (Go/Kotlin)                       │  │
│  │                                                                        │  │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────────────┐ │  │
│  │  │  Query Analyzer │  │  Hybrid Search  │  │   Graph Traversal      │ │  │
│  │  │  (intent, keys) │─▶│  Vector + Text  │─▶│   + Blast Radius       │ │  │
│  │  │                 │  │  + Graph paths  │  │                        │ │  │
│  │  └─────────────────┘  └─────────────────┘  └────────────────────────┘ │  │
│  │                                                    │                  │  │
│  │  ┌────────────────────────────────────────────────▼─────────────────┐ │  │
│  │  │                   Context Assembly                               │ │  │
│  │  │  • Rerank with cross-encoder                                     │ │  │
│  │  │  • Include parent context (class defs, imports)                  │ │  │
│  │  │  • Follow env var chains: EnvVar → Config → Code → UI            │ │  │
│  │  │  • Token budget: 32-64K optimal                                  │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                       │
│  ┌───────────────────────────────────▼───────────────────────────────────┐  │
│  │                    CLAUDE AGENTS LAYER (Python)                        │  │
│  │                                                                        │  │
│  │  ┌─────────────────────────────────────────────────────────────────┐  │  │
│  │  │                    Orchestrator Agent                           │  │  │
│  │  │   • Analyzes query intent                                       │  │  │
│  │  │   • Routes to specialized subagents                             │  │  │
│  │  │   • Synthesizes final answer                                    │  │  │
│  │  └─────────────────────────────────────────────────────────────────┘  │  │
│  │           │              │              │              │              │  │
│  │           ▼              ▼              ▼              ▼              │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────────┐  │  │
│  │  │  Explorer  │  │  Analyzer  │  │ Env Tracer │  │  Doc Writer    │  │  │
│  │  │  Subagent  │  │  Subagent  │  │  Subagent  │  │  Subagent      │  │  │
│  │  │            │  │            │  │            │  │                │  │  │
│  │  │ Read,Grep  │  │ Cypher     │  │ Graph path │  │ Generate docs  │  │  │
│  │  │ Glob       │  │ queries    │  │ traversal  │  │                │  │  │
│  │  └────────────┘  └────────────┘  └────────────┘  └────────────────┘  │  │
│  │                                                                        │  │
│  │  MCP Tools: neo4j_query, vector_search, file_read, grep, bash         │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                       │
│  ┌───────────────────────────────────▼───────────────────────────────────┐  │
│  │                      API LAYER (Go + Ktor)                             │  │
│  │   /api/repos    /api/index    /api/chat    /api/graph    /api/search  │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                       │
│  ┌───────────────────────────────────▼───────────────────────────────────┐  │
│  │                   FRONTEND (React + TypeScript + Vite)                 │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────────┐ │  │
│  │  │  Chat UI     │  │  Code Browser│  │  Graph Visualization         │ │  │
│  │  │  (streaming) │  │  (Monaco)    │  │  (Neo4j Bloom / neovis.js)   │ │  │
│  │  └──────────────┘  └──────────────┘  └──────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Модель данных Neo4j для Code Intelligence

### Node Labels (типы узлов)

```cypher
// Структурные узлы
(:Repository {name, url, lastIndexed, defaultBranch})
(:File {path, language, hash, size})
(:Package {name, path})  // Go packages, Python modules

// Code entities
(:Function {
  name, 
  signature, 
  docstring,
  startLine, endLine,
  complexity,          // cyclomatic complexity
  embedding,           // vector для semantic search
  nlDescription        // NL описание для embeddings
})

(:Class {name, docstring, embedding, nlDescription})
(:Method {name, signature, visibility, embedding})
(:Variable {name, type, scope})

// Config entities (для трассировки env → UI)
(:EnvVar {name, defaultValue, description})
(:ConfigFile {path, format})  // .env, yaml, properties
(:UIConfig {component, settingName, label})

// Git metadata
(:Commit {hash, message, author, timestamp})
(:Branch {name, isDefault})
```

### Relationship Types (типы связей)

```cypher
// Структурные связи
(Repository)-[:CONTAINS]->(File)
(File)-[:DECLARES]->(Function|Class)
(Package)-[:CONTAINS]->(File)

// Зависимости кода
(Function)-[:CALLS]->(Function)
(Function)-[:READS]->(Variable)
(Function)-[:WRITES]->(Variable)
(Class)-[:EXTENDS]->(Class)
(Class)-[:IMPLEMENTS]->(Interface)
(File)-[:IMPORTS]->(Package|File)

// Env var трассировка
(EnvVar)-[:DEFINED_IN]->(ConfigFile)
(EnvVar)-[:USED_IN]->(Function)
(EnvVar)-[:CONFIGURES]->(UIConfig)
(ConfigFile)-[:SETS]->(UIConfig)

// Git history
(Commit)-[:MODIFIES]->(File)
(Commit)-[:PARENT]->(Commit)
```

### Создание индексов

```cypher
// Vector index для semantic search
CREATE VECTOR INDEX `code-embeddings` 
FOR (n:Function) ON (n.embedding)
OPTIONS {indexConfig: {`vector.dimensions`: 1024, `vector.similarity_function`: 'cosine'}}

CREATE VECTOR INDEX `class-embeddings` 
FOR (n:Class) ON (n.embedding)
OPTIONS {indexConfig: {`vector.dimensions`: 1024, `vector.similarity_function`: 'cosine'}}

// Full-text index для keyword search
CREATE FULLTEXT INDEX code_fulltext 
FOR (n:Function|Class|Method) ON EACH [n.name, n.docstring, n.signature]

// B-tree indexes для быстрых lookups
CREATE INDEX file_path FOR (f:File) ON (f.path)
CREATE INDEX function_name FOR (f:Function) ON (f.name)
CREATE INDEX env_var_name FOR (e:EnvVar) ON (e.name)
```

---

## Cypher запросы для Code Intelligence

### 1. Blast Radius: что сломается при изменении функции

```cypher
// Найти все функции, зависящие от authenticate_user до 5 уровней
MATCH path = (target:Function {name: 'authenticate_user'})
              <-[:CALLS*1..5]-(caller:Function)
RETURN caller.name AS affected_function,
       length(path) AS distance,
       [node IN nodes(path) | node.name] AS call_chain
ORDER BY distance
```

### 2. Env Var Tracing: от переменной до UI

```cypher
// Полный путь DATABASE_URL через систему
MATCH (env:EnvVar {name: 'DATABASE_URL'})
OPTIONAL MATCH (env)-[:DEFINED_IN]->(config:ConfigFile)
OPTIONAL MATCH (env)-[:USED_IN]->(func:Function)
OPTIONAL MATCH (env)-[:CONFIGURES]->(ui:UIConfig)
RETURN env.name AS variable,
       config.path AS defined_in,
       collect(DISTINCT func.name) AS used_by_functions,
       collect(DISTINCT ui.settingName) AS affects_ui_settings
```

### 3. Hybrid Search: семантика + граф

```cypher
// Найти код похожий на "user authentication" и показать что он вызывает
CALL db.index.vector.queryNodes('code-embeddings', 20, $queryEmbedding)
YIELD node AS func, score
WHERE score > 0.75

// Расширить контекст графом
MATCH (func)-[:CALLS]->(called:Function)
MATCH (func)<-[:DECLARES]-(file:File)
RETURN func.name, func.signature, score,
       file.path,
       collect(called.name) AS calls
ORDER BY score DESC
LIMIT 10
```

### 4. Import Graph: зависимости модуля

```cypher
// Все транзитивные зависимости модуля auth
MATCH (f:File {path: 'src/auth/login.py'})
MATCH path = (f)-[:IMPORTS*1..10]->(dep:File)
RETURN DISTINCT dep.path AS dependency,
       length(path) AS depth
ORDER BY depth, dependency
```

### 5. Unused Code Detection

```cypher
// Публичные функции, которые никто не вызывает
MATCH (f:Function)
WHERE f.visibility = 'public'
  AND NOT exists((f)<-[:CALLS]-())
  AND NOT f.name STARTS WITH 'test_'
  AND NOT f.name STARTS WITH '__'
RETURN f.name, 
       [(f)<-[:DECLARES]-(file:File) | file.path][0] AS file
ORDER BY file
```

---

## Интеграция с jQAssistant (для Java/Kotlin)

**jQAssistant** — готовый инструмент для сканирования Java/Kotlin bytecode в Neo4j. Огромная экономия времени для JVM-проектов.

```bash
# Сканирование Java/Kotlin проекта
jqassistant scan -f java:classpath::build/classes
jqassistant scan -f kotlin:classpath::build/classes

# Запуск Neo4j сервера с данными
jqassistant server
```

**Готовые концепты из коробки:**
- Классы, методы, поля с полной метаинформацией
- Call graph (кто кого вызывает)
- Dependency graph (зависимости между артефактами)
- Аннотации (Spring, JPA, JAX-RS)
- Cyclomatic complexity

Для других языков (Go, Python, TypeScript) — собственный парсинг через Tree-sitter.

---

## MCP Server для Neo4j + Claude Agent SDK

```python
# mcp_neo4j_server.py
from mcp import MCPServer, Tool
from neo4j import GraphDatabase

class Neo4jMCPServer(MCPServer):
    def __init__(self, neo4j_uri: str, auth: tuple):
        self.driver = GraphDatabase.driver(neo4j_uri, auth=auth)
        
    @Tool(description="Execute a Cypher query on the code graph")
    async def cypher_query(self, query: str, params: dict = None) -> dict:
        """Run arbitrary Cypher query."""
        with self.driver.session() as session:
            result = session.run(query, params or {})
            return [record.data() for record in result]
    
    @Tool(description="Find functions similar to a description")
    async def semantic_search(self, description: str, limit: int = 10) -> list:
        """Semantic search over code embeddings."""
        embedding = await self.get_embedding(description)
        
        query = """
        CALL db.index.vector.queryNodes('code-embeddings', $limit, $embedding)
        YIELD node, score
        MATCH (node)<-[:DECLARES]-(f:File)
        RETURN node.name AS name, 
               node.signature AS signature,
               node.docstring AS docstring,
               f.path AS file,
               score
        ORDER BY score DESC
        """
        
        with self.driver.session() as session:
            result = session.run(query, {"limit": limit, "embedding": embedding})
            return [record.data() for record in result]
    
    @Tool(description="Trace env var usage through the codebase")
    async def trace_env_var(self, env_var_name: str) -> dict:
        """Trace how an environment variable flows through config and code."""
        query = """
        MATCH (env:EnvVar {name: $name})
        OPTIONAL MATCH (env)-[:DEFINED_IN]->(config:ConfigFile)
        OPTIONAL MATCH (env)-[:USED_IN]->(func:Function)<-[:DECLARES]-(file:File)
        OPTIONAL MATCH (env)-[:CONFIGURES]->(ui:UIConfig)
        RETURN env.name AS variable,
               env.defaultValue AS default_value,
               config.path AS config_file,
               collect(DISTINCT {function: func.name, file: file.path}) AS code_usage,
               collect(DISTINCT ui.settingName) AS ui_settings
        """
        
        with self.driver.session() as session:
            result = session.run(query, {"name": env_var_name})
            return result.single().data()
    
    @Tool(description="Get blast radius - what breaks if this function changes")
    async def blast_radius(self, function_name: str, max_depth: int = 5) -> list:
        """Find all code that depends on a function."""
        query = """
        MATCH (target:Function {name: $name})
        MATCH path = (target)<-[:CALLS*1..$depth]-(caller:Function)
        MATCH (caller)<-[:DECLARES]-(file:File)
        RETURN DISTINCT caller.name AS function,
               file.path AS file,
               length(path) AS distance
        ORDER BY distance, file
        """
        
        with self.driver.session() as session:
            result = session.run(query, {"name": function_name, "depth": max_depth})
            return [record.data() for record in result]
```

### Agent Configuration

```python
from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions

options = ClaudeAgentOptions(
    agents={
        'code-explorer': {
            'description': 'Explores codebase using graph queries',
            'tools': ['mcp__neo4j__cypher_query', 'mcp__neo4j__semantic_search'],
            'prompt': '''You explore code repositories using Neo4j graph database.
            Use cypher_query for structural questions (dependencies, calls).
            Use semantic_search for conceptual questions (find authentication code).'''
        },
        'env-tracer': {
            'description': 'Traces environment variables through config and code',
            'tools': ['mcp__neo4j__trace_env_var', 'mcp__neo4j__cypher_query'],
            'prompt': '''You trace how environment variables flow through the system:
            .env files → config parsers → application code → UI settings.
            Always show the complete chain.'''
        },
        'impact-analyzer': {
            'description': 'Analyzes impact of code changes',
            'tools': ['mcp__neo4j__blast_radius', 'mcp__neo4j__cypher_query'],
            'prompt': '''You analyze the impact of potential code changes.
            Show what other code depends on the target, ordered by distance.'''
        }
    },
    mcp_servers={
        'neo4j': {
            'type': 'stdio',
            'command': 'python mcp_neo4j_server.py'
        }
    }
)
```

---

## Indexing Pipeline: Go реализация

### Шаг 1: Парсинг AST с Tree-sitter

```go
package indexer

import (
    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/python"
    "github.com/smacker/go-tree-sitter/golang"
    "github.com/smacker/go-tree-sitter/typescript"
)

type CodeEntity struct {
    Type        string   // "function", "class", "method"
    Name        string
    Signature   string
    Docstring   string
    StartLine   int
    EndLine     int
    FilePath    string
    Calls       []string // функции которые вызывает
    Imports     []string
    Content     string   // raw code для NL generation
}

func ParseFile(content []byte, lang string) ([]CodeEntity, error) {
    parser := sitter.NewParser()
    
    switch lang {
    case "python":
        parser.SetLanguage(python.GetLanguage())
    case "go":
        parser.SetLanguage(golang.GetLanguage())
    case "typescript":
        parser.SetLanguage(typescript.GetLanguage())
    }
    
    tree := parser.Parse(nil, content)
    root := tree.RootNode()
    
    return extractEntities(root, content, lang), nil
}
```

### Шаг 2: Генерация NL-описаний

```python
import anthropic

client = anthropic.Anthropic()

def generate_nl_description(code_entity: dict) -> str:
    prompt = f"""Опиши что делает этот код в одном предложении на английском.
    
    Тип: {code_entity['type']}
    Имя: {code_entity['name']}
    Сигнатура: {code_entity['signature']}
    
    Код:
    ```
    {code_entity['content'][:1000]}
    ```
    
    Описание (одно предложение):"""
    
    response = client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=100,
        messages=[{"role": "user", "content": prompt}]
    )
    
    return response.content[0].text.strip()
```

### Шаг 3: Запись в Neo4j

```go
func WriteToNeo4j(ctx context.Context, driver neo4j.DriverWithContext, 
                  entities []CodeEntity, embeddings [][]float32) error {
    
    session := driver.NewSession(ctx, neo4j.SessionConfig{})
    defer session.Close(ctx)
    
    _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        for i, entity := range entities {
            _, err := tx.Run(ctx, `
                MERGE (f:File {path: $filePath})
                MERGE (e:Function {name: $name, file: $filePath})
                SET e.signature = $signature,
                    e.docstring = $docstring,
                    e.startLine = $startLine,
                    e.endLine = $endLine,
                    e.nlDescription = $nlDescription
                
                CALL db.create.setNodeVectorProperty(e, 'embedding', $embedding)
                
                MERGE (f)-[:DECLARES]->(e)
            `, map[string]any{
                "filePath":      entity.FilePath,
                "name":          entity.Name,
                "signature":     entity.Signature,
                "docstring":     entity.Docstring,
                "startLine":     entity.StartLine,
                "endLine":       entity.EndLine,
                "nlDescription": entity.NLDescription,
                "embedding":     embeddings[i],
            })
            if err != nil {
                return nil, err
            }
            
            // Создаём CALLS relationships
            for _, calledFunc := range entity.Calls {
                _, err := tx.Run(ctx, `
                    MATCH (caller:Function {name: $callerName, file: $callerFile})
                    MERGE (callee:Function {name: $calleeName})
                    MERGE (caller)-[:CALLS]->(callee)
                `, map[string]any{
                    "callerName": entity.Name,
                    "callerFile": entity.FilePath,
                    "calleeName": calledFunc,
                })
                if err != nil {
                    return nil, err
                }
            }
        }
        return nil, nil
    })
    
    return err
}
```

---

## Рекомендуемый стек технологий

| Компонент | Технология | Обоснование |
|-----------|------------|-------------|
| **Graph DB** | Neo4j 5.x+ (Community) | Бесплатен, vector search, Cypher |
| **Backend API** | Go (Fiber) + Kotlin (Ktor) | Твой стек, оба имеют Neo4j драйверы |
| **Парсинг кода** | Tree-sitter (Go bindings) | 100+ языков, быстрый |
| **Java/Kotlin scan** | jQAssistant | Готовый pipeline в Neo4j |
| **Embeddings** | voyage-code-3 API | SOTA для кода |
| **Embeddings fallback** | all-MiniLM-L6-v2 | Self-hosted, CPU |
| **Agent layer** | Claude Agent SDK (Python) | Subagents, MCP tools |
| **Frontend** | React + TypeScript + Vite | Твой стек |
| **Graph viz** | Neo4j Bloom или neovis.js | Интерактивная визуализация |

---

## MVP Timeline: 6-8 недель

| Неделя | Deliverable |
|--------|-------------|
| **1-2** | Git integration, Neo4j setup, базовая модель данных |
| **3-4** | Tree-sitter парсинг (Go, Python, TS), indexing pipeline |
| **5** | Vector search интеграция, NL descriptions |
| **6** | Claude Agent SDK, MCP server для Neo4j |
| **7** | React UI: chat + code browser |
| **8** | Graph visualization, polish, тестирование |

---

## Ключевые выводы

1. **Neo4j — правильный выбор** для code intelligence. Граф модель естественна для кода, встроенный vector search устраняет необходимость в отдельной vector DB.

2. **Hybrid queries** — киллер-фича: семантический поиск + граф траверсал в одном Cypher запросе.

3. **jQAssistant** для Java/Kotlin — огромная экономия времени, готовый pipeline.

4. **Embeddings нужны**, но для NL-описаний кода, не для raw code. Это повышает качество retrieval на ~12%.

5. **Cypher** — мощный язык для code analysis: blast radius, dependency chains, unused code detection — всё выражается элегантно.

6. **Neo4j Browser/Bloom** — бонус для команды: визуальное исследование кодовой базы без написания кода.
