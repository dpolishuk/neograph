package models

type File struct {
	ID       string `json:"id"`
	RepoID   string `json:"repoId"`
	Path     string `json:"path"`
	Language string `json:"language"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
}

// Language detection by extension
var LanguageByExtension = map[string]string{
	".go":   "go",
	".py":   "python",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".java": "java",
	".kt":   "kotlin",
	".kts":  "kotlin",
}

func DetectLanguage(path string) string {
	for ext, lang := range LanguageByExtension {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			return lang
		}
	}
	return ""
}
