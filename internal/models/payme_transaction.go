package models

import "github.com/google/uuid"

// PaymeTransaction stores Payme payment transaction state.
type PaymeTransaction struct {
	BaseModel
	TransactionID string     `gorm:"column:transaction_id;index" json:"transaction_id"`
	UserID        *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	OrderDetails  []byte     `gorm:"type:jsonb" json:"order_details"`
	Status        int        `json:"status"`
	Amount        int64      `json:"amount"`
	OrderID       string     `json:"order_id"`
	CreateTime    int64      `json:"create_time"`
	PerformTime   int64      `json:"perform_time"`
	CancelTime    int64      `json:"cancel_time"`
	Reason        *int       `json:"reason"`
	Provider      string     `json:"provider"`
	PrepareID     string     `json:"prepare_id"`
}
