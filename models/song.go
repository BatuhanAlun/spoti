package models

import "github.com/google/uuid"

// Song modeli, t_songs tablosunu temsil eder.
type Song struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Artist     string    `json:"artist"`
	Album      string    `json:"album"`
	Duration   int       `json:"duration"`
	ClickCount int       `json:"click_count"`
}
