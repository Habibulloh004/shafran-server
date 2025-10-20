package models

import (
	"time"

	"github.com/google/uuid"
)

type UserAddress struct {
	BaseModel
	UserID      uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	Label       string    `json:"label"`
	AddressLine string    `json:"address_line"`
	Apartment   string    `json:"apartment"`
	City        string    `json:"city"`
	District    string    `json:"district"`
	PostalCode  string    `json:"postal_code"`
	IsDefault   bool      `json:"is_default"`
}

type BonusTransaction struct {
	BaseModel
	UserID             uuid.UUID  `gorm:"type:uuid;index" json:"user_id"`
	TransactionNumber  string     `gorm:"uniqueIndex" json:"transaction_number"`
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	Amount             float64    `json:"amount"`
	Currency           string     `json:"currency"`
	OrderID            *uuid.UUID `gorm:"type:uuid" json:"order_id"`
	OccurredAt         time.Time  `json:"occurred_at"`
}

