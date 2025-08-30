package models

import "github.com/google/uuid"

type Coupon struct {
	ID     uuid.UUID `json:"id"`
	Code   string    `json:"code"`
	IsUsed bool      `json:"is_used"`
	UserID uuid.UUID `json:"user_id"`
}
