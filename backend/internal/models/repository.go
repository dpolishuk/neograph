package models

import "time"

type Repository struct {
	ID             string    `json:"id"`
	URL            string    `json:"url"`
	Name           string    `json:"name"`
	DefaultBranch  string    `json:"defaultBranch"`
	LastIndexed    time.Time `json:"lastIndexed"`
	Status         string    `json:"status"` // pending, indexing, ready, error
	FilesCount     int       `json:"filesCount"`
	FunctionsCount int       `json:"functionsCount"`
}

type CreateRepositoryInput struct {
	URL    string `json:"url" validate:"required,url"`
	Branch string `json:"branch"`
}
