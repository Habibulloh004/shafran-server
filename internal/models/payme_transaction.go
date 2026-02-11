package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymeTransaction stores Payme payment transaction state.
type PaymeTransaction struct {
	BaseModel
	TransactionID    string     `gorm:"column:transaction_id;index" json:"transaction_id"`
	UserID           *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	OrderDetails     []byte     `gorm:"type:jsonb" json:"order_details"`
	Status           int        `json:"status"`
	Amount           int64      `json:"amount"`
	OrderID          string     `json:"order_id"`
	CreateTime       int64      `json:"create_time"`
	PerformTime      int64      `json:"perform_time"`
	CancelTime       int64      `json:"cancel_time"`
	Reason           *int       `json:"reason"`
	Provider         string     `json:"provider"`
	PrepareID        string     `json:"prepare_id"`
	BillzOrderID     string     `gorm:"column:billz_order_id" json:"billz_order_id"`
	BillzOrderNumber string     `gorm:"column:billz_order_number" json:"billz_order_number"`
	BillzOrderType   string     `gorm:"column:billz_order_type" json:"billz_order_type"`
	BillzSyncedAt    *time.Time `gorm:"column:billz_synced_at" json:"billz_synced_at"`
	BillzSyncError   string     `gorm:"column:billz_sync_error" json:"billz_sync_error"`
}
