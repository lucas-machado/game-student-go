package model

import "time"

type Card struct {
	ID                int
	UserID            int
	StripePayMethodID string
	CreatedAt         time.Time
}
