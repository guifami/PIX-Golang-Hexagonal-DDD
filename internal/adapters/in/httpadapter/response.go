package httpadapter

import "time"

// ErrorResponse is the standard error envelope returned by all handlers.
type ErrorResponse struct {
	Message string `json:"message"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
