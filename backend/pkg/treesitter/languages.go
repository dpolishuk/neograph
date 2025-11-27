package treesitter

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

var languages = map[string]*sitter.Language{
	"go":         golang.GetLanguage(),
	"python":     python.GetLanguage(),
	"typescript": typescript.GetLanguage(),
	"javascript": javascript.GetLanguage(),
	"java":       java.GetLanguage(),
	"kotlin":     kotlin.GetLanguage(),
}

func GetLanguage(name string) *sitter.Language {
	return languages[name]
}

func SupportedLanguages() []string {
	keys := make([]string, 0, len(languages))
	for k := range languages {
		keys = append(keys, k)
	}
	return keys
}
