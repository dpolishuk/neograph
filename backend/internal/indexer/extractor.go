package indexer

import (
	"context"
	"fmt"
	"strings"

	"github.com/dpolishuk/neograph/backend/pkg/treesitter"
	sitter "github.com/smacker/go-tree-sitter"
)

// CodeEntity represents a code entity (function, class, method, etc.)
type CodeEntity struct {
	Type      string   // "function", "method", "class", "interface", etc.
	Name      string   // Name of the entity
	Signature string   // Full signature (e.g., "func myFunc(a int) error")
	Docstring string   // Documentation/comments
	StartLine int      // Starting line number (1-indexed)
	EndLine   int      // Ending line number (1-indexed)
	FilePath  string   // Path to the source file
	Calls     []string // List of function/method calls made within this entity
	Content   string   // Full source code content of the entity
}

// Extractor wraps the tree-sitter parser for code entity extraction
type Extractor struct {
	parser *treesitter.Parser
}

// NewExtractor creates a new code entity extractor
func NewExtractor() *Extractor {
	return &Extractor{
		parser: treesitter.NewParser(),
	}
}

// Close releases resources used by the extractor
func (e *Extractor) Close() {
	e.parser.Close()
}

// Extract extracts code entities from the given source code
func (e *Extractor) Extract(ctx context.Context, content []byte, language string, filePath string) ([]CodeEntity, error) {
	tree, err := e.parser.Parse(ctx, content, language)
	if err != nil {
		return nil, fmt.Errorf("failed to parse code: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()

	switch language {
	case "go":
		return e.extractGo(root, content, filePath), nil
	case "python":
		return e.extractPython(root, content, filePath), nil
	case "typescript", "javascript":
		return e.extractTypeScript(root, content, filePath), nil
	case "java":
		return e.extractJava(root, content, filePath), nil
	case "kotlin":
		return e.extractKotlin(root, content, filePath), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// extractGo extracts entities from Go code
func (e *Extractor) extractGo(root *sitter.Node, content []byte, filePath string) []CodeEntity {
	var entities []CodeEntity
	e.traverseNode(root, content, func(node *sitter.Node) {
		nodeType := node.Type()

		switch nodeType {
		case "function_declaration":
			entity := e.extractGoFunction(node, content, filePath, "function")
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "method_declaration":
			entity := e.extractGoFunction(node, content, filePath, "method")
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "type_declaration":
			// Extract struct declarations
			// Look for struct_type within the type_declaration
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				if child != nil && child.Type() == "type_spec" {
					// Check if this type spec contains a struct_type
					for j := 0; j < int(child.NamedChildCount()); j++ {
						typeChild := child.NamedChild(j)
						if typeChild != nil && typeChild.Type() == "struct_type" {
							entity := e.extractGoStruct(node, child, content, filePath)
							if entity != nil {
								entities = append(entities, *entity)
							}
							break
						}
					}
				}
			}
		}
	})
	return entities
}

// extractGoFunction extracts a Go function or method
func (e *Extractor) extractGoFunction(node *sitter.Node, content []byte, filePath string, entityType string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := getNodeContent(node, content)
	docstring := getPrecedingComment(node, content)
	calls := extractCalls(node, content)

	return &CodeEntity{
		Type:      entityType,
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   signature,
	}
}

// extractGoStruct extracts a Go struct declaration
func (e *Extractor) extractGoStruct(declNode *sitter.Node, typeSpec *sitter.Node, content []byte, filePath string) *CodeEntity {
	// Find the type_identifier within the type_spec
	var nameNode *sitter.Node
	for i := 0; i < int(typeSpec.NamedChildCount()); i++ {
		child := typeSpec.NamedChild(i)
		if child != nil && child.Type() == "type_identifier" {
			nameNode = child
			break
		}
	}

	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := getNodeContent(declNode, content)
	docstring := getPrecedingComment(declNode, content)

	return &CodeEntity{
		Type:      "class",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(declNode.StartPoint().Row) + 1,
		EndLine:   int(declNode.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     []string{},
		Content:   signature,
	}
}

// extractPython extracts entities from Python code
func (e *Extractor) extractPython(root *sitter.Node, content []byte, filePath string) []CodeEntity {
	var entities []CodeEntity
	e.traverseNode(root, content, func(node *sitter.Node) {
		nodeType := node.Type()

		switch nodeType {
		case "function_definition":
			// Check if this is a method (has a parent class)
			isMethod := false
			parent := node.Parent()
			for parent != nil {
				if parent.Type() == "class_definition" {
					isMethod = true
					break
				}
				parent = parent.Parent()
			}

			entityType := "function"
			if isMethod {
				entityType = "method"
			}

			entity := e.extractPythonFunction(node, content, filePath, entityType)
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "class_definition":
			entity := e.extractPythonClass(node, content, filePath)
			if entity != nil {
				entities = append(entities, *entity)
			}
		}
	})
	return entities
}

// extractPythonFunction extracts a Python function or method
func (e *Extractor) extractPythonFunction(node *sitter.Node, content []byte, filePath string, entityType string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getPythonSignature(node, content)
	docstring := getPythonDocstring(node, content)
	calls := extractCalls(node, content)

	return &CodeEntity{
		Type:      entityType,
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   getNodeContent(node, content),
	}
}

// extractPythonClass extracts a Python class
func (e *Extractor) extractPythonClass(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getPythonSignature(node, content)
	docstring := getPythonDocstring(node, content)

	return &CodeEntity{
		Type:      "class",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     []string{},
		Content:   getNodeContent(node, content),
	}
}

// getPythonSignature extracts just the signature line (def/class line) from Python code
func (e *Extractor) getPythonSignature(node *sitter.Node, content []byte) string {
	fullContent := getNodeContent(node, content)
	lines := strings.Split(fullContent, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return fullContent
}

// extractTypeScript extracts entities from TypeScript/JavaScript code
func (e *Extractor) extractTypeScript(root *sitter.Node, content []byte, filePath string) []CodeEntity {
	var entities []CodeEntity
	e.traverseNode(root, content, func(node *sitter.Node) {
		nodeType := node.Type()

		switch nodeType {
		case "function_declaration":
			entity := e.extractTSFunction(node, content, filePath, "function")
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "class_declaration":
			entity := e.extractTSClass(node, content, filePath)
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "method_definition":
			entity := e.extractTSMethod(node, content, filePath)
			if entity != nil {
				entities = append(entities, *entity)
			}
		}
	})
	return entities
}

// extractTSFunction extracts a TypeScript/JavaScript function
func (e *Extractor) extractTSFunction(node *sitter.Node, content []byte, filePath string, entityType string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getTSSignature(node, content)
	docstring := getPrecedingComment(node, content)
	calls := extractCalls(node, content)

	return &CodeEntity{
		Type:      entityType,
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   getNodeContent(node, content),
	}
}

// extractTSClass extracts a TypeScript/JavaScript class
func (e *Extractor) extractTSClass(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getTSSignature(node, content)
	docstring := getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      "class",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     []string{},
		Content:   getNodeContent(node, content),
	}
}

// extractTSMethod extracts a TypeScript/JavaScript method
func (e *Extractor) extractTSMethod(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getTSSignature(node, content)
	docstring := getPrecedingComment(node, content)
	calls := extractCalls(node, content)

	return &CodeEntity{
		Type:      "method",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   getNodeContent(node, content),
	}
}

// getTSSignature extracts the signature line from TypeScript/JavaScript code
func (e *Extractor) getTSSignature(node *sitter.Node, content []byte) string {
	fullContent := getNodeContent(node, content)

	// For functions and methods, try to extract just the signature
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		// Get content up to the body
		startByte := node.StartByte()
		endByte := bodyNode.StartByte()
		if endByte > startByte {
			sig := string(content[startByte:endByte])
			return strings.TrimSpace(sig)
		}
	}

	// Fallback to first line
	lines := strings.Split(fullContent, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return fullContent
}

// extractJava extracts entities from Java code
func (e *Extractor) extractJava(root *sitter.Node, content []byte, filePath string) []CodeEntity {
	var entities []CodeEntity
	e.traverseNode(root, content, func(node *sitter.Node) {
		nodeType := node.Type()

		switch nodeType {
		case "method_declaration":
			entity := e.extractJavaMethod(node, content, filePath)
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "class_declaration":
			entity := e.extractJavaClass(node, content, filePath, "class")
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "interface_declaration":
			entity := e.extractJavaClass(node, content, filePath, "interface")
			if entity != nil {
				entities = append(entities, *entity)
			}
		}
	})
	return entities
}

// extractJavaMethod extracts a Java method
func (e *Extractor) extractJavaMethod(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getJavaSignature(node, content)
	docstring := getPrecedingComment(node, content)
	calls := extractCalls(node, content)

	return &CodeEntity{
		Type:      "method",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   getNodeContent(node, content),
	}
}

// extractJavaClass extracts a Java class or interface
func (e *Extractor) extractJavaClass(node *sitter.Node, content []byte, filePath string, entityType string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getJavaSignature(node, content)
	docstring := getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      entityType,
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     []string{},
		Content:   getNodeContent(node, content),
	}
}

// getJavaSignature extracts the signature from Java code
func (e *Extractor) getJavaSignature(node *sitter.Node, content []byte) string {
	fullContent := getNodeContent(node, content)

	// For methods and classes, try to extract just the signature
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		// Get content up to the body
		startByte := node.StartByte()
		endByte := bodyNode.StartByte()
		if endByte > startByte {
			sig := string(content[startByte:endByte])
			return strings.TrimSpace(sig)
		}
	}

	// Fallback to first line
	lines := strings.Split(fullContent, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return fullContent
}

// extractKotlin extracts entities from Kotlin code
func (e *Extractor) extractKotlin(root *sitter.Node, content []byte, filePath string) []CodeEntity {
	var entities []CodeEntity
	e.traverseNode(root, content, func(node *sitter.Node) {
		nodeType := node.Type()

		switch nodeType {
		case "function_declaration":
			entity := e.extractKotlinFunction(node, content, filePath)
			if entity != nil {
				entities = append(entities, *entity)
			}
		case "class_declaration":
			entity := e.extractKotlinClass(node, content, filePath)
			if entity != nil {
				entities = append(entities, *entity)
			}
		}
	})
	return entities
}

// extractKotlinFunction extracts a Kotlin function
func (e *Extractor) extractKotlinFunction(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	// Find simple_identifier child
	var nameNode *sitter.Node
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Type() == "simple_identifier" {
			nameNode = child
			break
		}
	}

	if nameNode == nil {
		return nil
	}

	// Check if this is a method (inside a class)
	entityType := "function"
	parent := node.Parent()
	for parent != nil {
		if parent.Type() == "class_declaration" || parent.Type() == "class_body" {
			entityType = "method"
			break
		}
		parent = parent.Parent()
	}

	name := getNodeContent(nameNode, content)
	signature := e.getKotlinSignature(node, content)
	docstring := getPrecedingComment(node, content)
	calls := extractCalls(node, content)

	return &CodeEntity{
		Type:      entityType,
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   getNodeContent(node, content),
	}
}

// extractKotlinClass extracts a Kotlin class
func (e *Extractor) extractKotlinClass(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	// Find type_identifier child
	var nameNode *sitter.Node
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Type() == "type_identifier" {
			nameNode = child
			break
		}
	}

	if nameNode == nil {
		return nil
	}

	name := getNodeContent(nameNode, content)
	signature := e.getKotlinSignature(node, content)
	docstring := getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      "class",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     []string{},
		Content:   getNodeContent(node, content),
	}
}

// getKotlinSignature extracts the signature from Kotlin code
func (e *Extractor) getKotlinSignature(node *sitter.Node, content []byte) string {
	fullContent := getNodeContent(node, content)

	// For functions, try to extract just the signature (before the body)
	bodyNode := node.ChildByFieldName("function_body")
	if bodyNode != nil {
		startByte := node.StartByte()
		endByte := bodyNode.StartByte()
		if endByte > startByte {
			sig := string(content[startByte:endByte])
			return strings.TrimSpace(sig)
		}
	}

	// For classes, extract up to the class body
	classBodyNode := node.ChildByFieldName("class_body")
	if classBodyNode != nil {
		startByte := node.StartByte()
		endByte := classBodyNode.StartByte()
		if endByte > startByte {
			sig := string(content[startByte:endByte])
			return strings.TrimSpace(sig)
		}
	}

	// Fallback to first line
	lines := strings.Split(fullContent, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return fullContent
}

// Helper functions

// traverseNode recursively traverses the AST and calls the callback for each node
func (e *Extractor) traverseNode(node *sitter.Node, content []byte, callback func(*sitter.Node)) {
	if node == nil {
		return
	}

	callback(node)

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		e.traverseNode(child, content, callback)
	}
}

// getNodeContent extracts the text content of a node
func getNodeContent(node *sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	return node.Content(content)
}

// getPrecedingComment extracts comment/docstring from the node immediately preceding the given node
func getPrecedingComment(node *sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}

	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	prevType := prev.Type()
	if prevType == "comment" || prevType == "line_comment" || prevType == "block_comment" {
		comment := getNodeContent(prev, content)
		// Clean up comment markers
		comment = strings.TrimPrefix(comment, "//")
		comment = strings.TrimPrefix(comment, "/*")
		comment = strings.TrimSuffix(comment, "*/")
		comment = strings.TrimPrefix(comment, "#")
		return strings.TrimSpace(comment)
	}

	return ""
}

// getPythonDocstring extracts the Python docstring from a function or class
func getPythonDocstring(node *sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}

	// Look for the body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil || bodyNode.Type() != "block" {
		return ""
	}

	// Get the first child of the body (after the colon)
	if bodyNode.NamedChildCount() == 0 {
		return ""
	}

	firstChild := bodyNode.NamedChild(0)
	if firstChild == nil {
		return ""
	}

	// Check if it's an expression statement containing a string
	if firstChild.Type() == "expression_statement" {
		if firstChild.NamedChildCount() > 0 {
			exprChild := firstChild.NamedChild(0)
			if exprChild != nil && exprChild.Type() == "string" {
				docstring := getNodeContent(exprChild, content)
				// Remove the quotes
				docstring = strings.Trim(docstring, "\"'")
				return strings.TrimSpace(docstring)
			}
		}
	}

	return ""
}

// extractCalls extracts function/method calls within a node
func extractCalls(node *sitter.Node, content []byte) []string {
	var calls []string
	callsMap := make(map[string]bool) // To avoid duplicates

	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		if n == nil {
			return
		}

		nodeType := n.Type()
		if nodeType == "call_expression" || nodeType == "call" {
			// Get the function being called
			funcNode := n.ChildByFieldName("function")
			if funcNode != nil {
				callName := getNodeContent(funcNode, content)
				if callName != "" && !callsMap[callName] {
					calls = append(calls, callName)
					callsMap[callName] = true
				}
			}
		}

		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			traverse(child)
		}
	}

	traverse(node)
	return calls
}
