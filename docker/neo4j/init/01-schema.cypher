// Constraints
CREATE CONSTRAINT repo_url IF NOT EXISTS FOR (r:Repository) REQUIRE r.url IS UNIQUE;
CREATE CONSTRAINT file_path IF NOT EXISTS FOR (f:File) REQUIRE (f.repoId, f.path) IS UNIQUE;

// Indexes for lookups
CREATE INDEX file_language IF NOT EXISTS FOR (f:File) ON (f.language);
CREATE INDEX function_name IF NOT EXISTS FOR (fn:Function) ON (fn.name);
CREATE INDEX class_name IF NOT EXISTS FOR (c:Class) ON (c.name);
CREATE INDEX method_name IF NOT EXISTS FOR (m:Method) ON (m.name);

// Full-text index for code search
CREATE FULLTEXT INDEX code_fulltext IF NOT EXISTS
FOR (n:Function|Class|Method) ON EACH [n.name, n.docstring, n.signature];
