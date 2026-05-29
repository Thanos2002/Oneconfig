package config

import "fmt"

// ErrorType identifies the category of config error.
type ErrorType int

const (
	ErrNotFound         ErrorType = iota // Config file not found
	ErrInvalidYAML                       // YAML syntax error
	ErrSchemaValidation                  // JSON Schema validation failure
	ErrInvalidConfig                     // Semantic validation failure
)

// ConfigError is a structured error with a user-friendly message and fix hint.
type ConfigError struct {
	Type    ErrorType
	Message string
	Hint    string
}

func (e *ConfigError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s\n\n  💡 %s", e.Message, e.Hint)
	}
	return e.Message
}
