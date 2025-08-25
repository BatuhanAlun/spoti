package models

import "github.com/google/uuid"

// Playlist modeli, t_playlist tablosunu temsil eder.
type Playlist struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	UserID uuid.UUID `json:"user_id"`
}

// PlaylistSong modeli, t_playlist_songs tablosunu temsil eder.
type PlaylistSong struct {
	PlaylistID uuid.UUID `json:"playlist_id"`
	SongID     uuid.UUID `json:"song_id"`
}
