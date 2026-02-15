package domain

import "time"

type Account struct {
	ID          string
	Email       string
	Provider    string
	DisplayName string
	CreatedAt   time.Time
}
