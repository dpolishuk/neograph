package models

type CodeEntityType string

const (
	EntityFunction CodeEntityType = "Function"
	EntityClass    CodeEntityType = "Class"
	EntityMethod   CodeEntityType = "Method"
	EntityVariable CodeEntityType = "Variable"
)

type CodeEntity struct {
	ID        string         `json:"id"`
	Type      CodeEntityType `json:"type"`
	Name      string         `json:"name"`
	Signature string         `json:"signature,omitempty"`
	Docstring string         `json:"docstring,omitempty"`
	StartLine int            `json:"startLine"`
	EndLine   int            `json:"endLine"`
	FilePath  string         `json:"filePath"`
	FileID    string         `json:"fileId"`
	RepoID    string         `json:"repoId"`
	Content   string         `json:"content,omitempty"`

	// For embeddings
	NLDescription string    `json:"nlDescription,omitempty"`
	Embedding     []float32 `json:"embedding,omitempty"`

	// Relationships (populated on query)
	Calls   []string `json:"calls,omitempty"`
	Imports []string `json:"imports,omitempty"`
}

type CallRelation struct {
	CallerID string `json:"callerId"`
	CalleeID string `json:"calleeId"`
	Line     int    `json:"line"`
}

type ImportRelation struct {
	FileID     string `json:"fileId"`
	ImportPath string `json:"importPath"`
	Alias      string `json:"alias,omitempty"`
}
