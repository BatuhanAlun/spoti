package models

import "github.com/google/uuid"

// Role modeli, t_roles tablosunu temsil eder.
type Role struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// User modeli, t_users tablosunu temsil eder.
type User struct {
	ID        uuid.UUID `json:"id"`
	RoleID    uuid.UUID `json:"role_id"`
	HesapTuru string    `json:"hesap_turu"`
	Cash      float64   `json:"cash"`
}
