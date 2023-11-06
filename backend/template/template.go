package template

// Template represents a template in the system
type Template struct {
	ID     int    `json:"id"`
	Prompt string `json:"prompt"`
}

type Database interface {
	GetTemplates() ([]Template, error)
	AddTemplate(prompt string) error
	DeleteTemplate(id int) error
}
