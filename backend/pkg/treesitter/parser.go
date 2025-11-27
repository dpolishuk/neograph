package treesitter

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

type Parser struct {
	parser *sitter.Parser
}

func NewParser() *Parser {
	return &Parser{
		parser: sitter.NewParser(),
	}
}

func (p *Parser) Parse(ctx context.Context, content []byte, language string) (*sitter.Tree, error) {
	lang := GetLanguage(language)
	if lang == nil {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	p.parser.SetLanguage(lang)

	tree, err := p.parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return tree, nil
}

func (p *Parser) Close() {
	p.parser.Close()
}
