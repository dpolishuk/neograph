package indexer

import (
	"context"
	"testing"
)

func TestExtractGoFunctions(t *testing.T) {
	extractor := NewExtractor()
	defer extractor.Close()

	goCode := `package main

// Add adds two numbers together
func Add(a, b int) int {
	return a + b
}

// Calculator is a simple calculator
type Calculator struct {
	result int
}

// Multiply multiplies two numbers
func (c *Calculator) Multiply(a, b int) int {
	result := a * b
	c.result = result
	return result
}
`

	ctx := context.Background()
	entities, err := extractor.Extract(ctx, []byte(goCode), "go", "test.go")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entities) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(entities))
	}

	// Check Add function
	found := false
	for _, entity := range entities {
		if entity.Name == "Add" {
			found = true
			if entity.Type != "function" {
				t.Errorf("Expected type 'function', got '%s'", entity.Type)
			}
			if entity.Docstring != "Add adds two numbers together" {
				t.Errorf("Expected docstring 'Add adds two numbers together', got '%s'", entity.Docstring)
			}
			if entity.StartLine != 4 {
				t.Errorf("Expected start line 4, got %d", entity.StartLine)
			}
			if entity.FilePath != "test.go" {
				t.Errorf("Expected file path 'test.go', got '%s'", entity.FilePath)
			}
		}
	}
	if !found {
		t.Error("Add function not found in entities")
	}

	// Check Calculator struct
	found = false
	for _, entity := range entities {
		if entity.Name == "Calculator" {
			found = true
			if entity.Type != "class" {
				t.Errorf("Expected type 'class', got '%s'", entity.Type)
			}
			if entity.Docstring != "Calculator is a simple calculator" {
				t.Errorf("Expected docstring 'Calculator is a simple calculator', got '%s'", entity.Docstring)
			}
		}
	}
	if !found {
		t.Error("Calculator struct not found in entities")
	}

	// Check Multiply method
	found = false
	for _, entity := range entities {
		if entity.Name == "Multiply" {
			found = true
			if entity.Type != "method" {
				t.Errorf("Expected type 'method', got '%s'", entity.Type)
			}
			if entity.Docstring != "Multiply multiplies two numbers" {
				t.Errorf("Expected docstring 'Multiply multiplies two numbers', got '%s'", entity.Docstring)
			}
		}
	}
	if !found {
		t.Error("Multiply method not found in entities")
	}
}

func TestExtractPythonFunctions(t *testing.T) {
	extractor := NewExtractor()
	defer extractor.Close()

	pythonCode := `def add(a, b):
    """Add two numbers together"""
    return a + b

class Calculator:
    """A simple calculator class"""

    def __init__(self):
        self.result = 0

    def multiply(self, a, b):
        """Multiply two numbers"""
        result = a * b
        self.result = result
        return result
`

	ctx := context.Background()
	entities, err := extractor.Extract(ctx, []byte(pythonCode), "python", "test.py")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entities) != 4 {
		t.Errorf("Expected 4 entities, got %d", len(entities))
	}

	// Check add function
	found := false
	for _, entity := range entities {
		if entity.Name == "add" {
			found = true
			if entity.Type != "function" {
				t.Errorf("Expected type 'function', got '%s'", entity.Type)
			}
			if entity.Docstring != "Add two numbers together" {
				t.Errorf("Expected docstring 'Add two numbers together', got '%s'", entity.Docstring)
			}
			if entity.StartLine != 1 {
				t.Errorf("Expected start line 1, got %d", entity.StartLine)
			}
		}
	}
	if !found {
		t.Error("add function not found in entities")
	}

	// Check Calculator class
	found = false
	for _, entity := range entities {
		if entity.Name == "Calculator" {
			found = true
			if entity.Type != "class" {
				t.Errorf("Expected type 'class', got '%s'", entity.Type)
			}
			if entity.Docstring != "A simple calculator class" {
				t.Errorf("Expected docstring 'A simple calculator class', got '%s'", entity.Docstring)
			}
		}
	}
	if !found {
		t.Error("Calculator class not found in entities")
	}

	// Check __init__ method
	found = false
	for _, entity := range entities {
		if entity.Name == "__init__" {
			found = true
			if entity.Type != "method" {
				t.Errorf("Expected type 'method', got '%s'", entity.Type)
			}
		}
	}
	if !found {
		t.Error("__init__ method not found in entities")
	}

	// Check multiply method
	found = false
	for _, entity := range entities {
		if entity.Name == "multiply" {
			found = true
			if entity.Type != "method" {
				t.Errorf("Expected type 'method', got '%s'", entity.Type)
			}
			if entity.Docstring != "Multiply two numbers" {
				t.Errorf("Expected docstring 'Multiply two numbers', got '%s'", entity.Docstring)
			}
		}
	}
	if !found {
		t.Error("multiply method not found in entities")
	}
}

func TestExtractTypeScript(t *testing.T) {
	extractor := NewExtractor()
	defer extractor.Close()

	tsCode := `// Add two numbers
function add(a: number, b: number): number {
    return a + b;
}

class Calculator {
    result: number = 0;

    multiply(a: number, b: number): number {
        this.result = a * b;
        return this.result;
    }
}
`

	ctx := context.Background()
	entities, err := extractor.Extract(ctx, []byte(tsCode), "typescript", "test.ts")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entities) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(entities))
	}

	// Check for function
	foundFunc := false
	for _, entity := range entities {
		if entity.Name == "add" && entity.Type == "function" {
			foundFunc = true
		}
	}
	if !foundFunc {
		t.Error("add function not found")
	}

	// Check for class
	foundClass := false
	for _, entity := range entities {
		if entity.Name == "Calculator" && entity.Type == "class" {
			foundClass = true
		}
	}
	if !foundClass {
		t.Error("Calculator class not found")
	}

	// Check for method
	foundMethod := false
	for _, entity := range entities {
		if entity.Name == "multiply" && entity.Type == "method" {
			foundMethod = true
		}
	}
	if !foundMethod {
		t.Error("multiply method not found")
	}
}

func TestExtractJava(t *testing.T) {
	extractor := NewExtractor()
	defer extractor.Close()

	javaCode := `public class Calculator {
    private int result;

    public int add(int a, int b) {
        return a + b;
    }

    public int multiply(int a, int b) {
        result = a * b;
        return result;
    }
}

interface Operation {
    int execute(int a, int b);
}
`

	ctx := context.Background()
	entities, err := extractor.Extract(ctx, []byte(javaCode), "java", "test.java")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entities) < 3 {
		t.Errorf("Expected at least 3 entities, got %d", len(entities))
	}

	// Check for class
	foundClass := false
	for _, entity := range entities {
		if entity.Name == "Calculator" && entity.Type == "class" {
			foundClass = true
		}
	}
	if !foundClass {
		t.Error("Calculator class not found")
	}

	// Check for interface
	foundInterface := false
	for _, entity := range entities {
		if entity.Name == "Operation" && entity.Type == "interface" {
			foundInterface = true
		}
	}
	if !foundInterface {
		t.Error("Operation interface not found")
	}

	// Check for methods
	foundAdd := false
	foundMultiply := false
	for _, entity := range entities {
		if entity.Name == "add" && entity.Type == "method" {
			foundAdd = true
		}
		if entity.Name == "multiply" && entity.Type == "method" {
			foundMultiply = true
		}
	}
	if !foundAdd {
		t.Error("add method not found")
	}
	if !foundMultiply {
		t.Error("multiply method not found")
	}
}

func TestExtractKotlin(t *testing.T) {
	extractor := NewExtractor()
	defer extractor.Close()

	kotlinCode := `fun add(a: Int, b: Int): Int {
    return a + b
}

class Calculator {
    var result: Int = 0

    fun multiply(a: Int, b: Int): Int {
        result = a * b
        return result
    }
}
`

	ctx := context.Background()
	entities, err := extractor.Extract(ctx, []byte(kotlinCode), "kotlin", "test.kt")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(entities) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(entities))
	}

	// Check for function
	foundFunc := false
	for _, entity := range entities {
		if entity.Name == "add" && entity.Type == "function" {
			foundFunc = true
		}
	}
	if !foundFunc {
		t.Error("add function not found")
	}

	// Check for class
	foundClass := false
	for _, entity := range entities {
		if entity.Name == "Calculator" && entity.Type == "class" {
			foundClass = true
		}
	}
	if !foundClass {
		t.Error("Calculator class not found")
	}

	// Check for method
	foundMethod := false
	for _, entity := range entities {
		if entity.Name == "multiply" && entity.Type == "method" {
			foundMethod = true
		}
	}
	if !foundMethod {
		t.Error("multiply method not found")
	}
}

func TestExtractCalls(t *testing.T) {
	extractor := NewExtractor()
	defer extractor.Close()

	goCode := `package main

func helper() int {
	return 42
}

func process(x int) int {
	return x * 2
}

func main() {
	result := helper()
	final := process(result)
	println(final)
}
`

	ctx := context.Background()
	entities, err := extractor.Extract(ctx, []byte(goCode), "go", "test.go")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Find main function
	var mainFunc *CodeEntity
	for i := range entities {
		if entities[i].Name == "main" {
			mainFunc = &entities[i]
			break
		}
	}

	if mainFunc == nil {
		t.Fatal("main function not found")
	}

	if len(mainFunc.Calls) < 2 {
		t.Errorf("Expected at least 2 calls in main, got %d", len(mainFunc.Calls))
	}

	// Check if helper and process are in the calls
	foundHelper := false
	foundProcess := false
	for _, call := range mainFunc.Calls {
		if call == "helper" {
			foundHelper = true
		}
		if call == "process" {
			foundProcess = true
		}
	}

	if !foundHelper {
		t.Error("helper call not found in main function")
	}
	if !foundProcess {
		t.Error("process call not found in main function")
	}
}

func TestUnsupportedLanguage(t *testing.T) {
	extractor := NewExtractor()
	defer extractor.Close()

	ctx := context.Background()
	_, err := extractor.Extract(ctx, []byte("test"), "unsupported", "test.txt")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}
