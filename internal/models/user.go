package models

import (
	"time"
)

// User represents an authenticated customer.
type User struct {
	BaseModel
	FirstName        string             `json:"first_name"`
	LastName         string             `json:"last_name"`
	Phone            string             `gorm:"uniqueIndex" json:"phone"`
	DisplayName      string             `json:"display_name"`
	PasswordHash     string             `json:"-"`
	IsVerified       bool               `json:"is_verified"`
	Addresses        []UserAddress      `json:"addresses,omitempty"`
	BonusTransactions []BonusTransaction `json:"bonus_transactions,omitempty"`
	Orders           []Order            `json:"orders,omitempty"`
}

// SMSVerification keeps track of OTP codes sent to users.
type SMSVerification struct {
	BaseModel
	Phone     string    `gorm:"index" json:"phone"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	Verified  bool      `json:"verified"`
	UsedAt    *time.Time `json:"used_at"`
}

// PasswordResetToken stores tokens for the forgot-password flow.
type PasswordResetToken struct {
	BaseModel
	Phone     string     `gorm:"index" json:"phone"`
	Token     string     `gorm:"uniqueIndex" json:"token"`
	Code      string     `json:"code"`
	SessionID string     `json:"session_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	Verified  bool       `json:"verified"`
	UsedAt    *time.Time `json:"used_at"`
}

