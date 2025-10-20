package models

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	BaseModel
	UserID           uuid.UUID  `gorm:"type:uuid;index" json:"user_id"`
	User             *User      `json:"user,omitempty"`
	OrderNumber      string     `gorm:"uniqueIndex" json:"order_number"`
	Status           string     `json:"status"`
	PlacedAt         time.Time  `json:"placed_at"`
	Subtotal         float64    `json:"subtotal"`
	ShippingFee      float64    `json:"shipping_fee"`
	TotalAmount      float64    `json:"total_amount"`
	Currency         string     `json:"currency"`
	DeliveryMethod   string     `json:"delivery_method"`
	DeliveryAddressID *uuid.UUID `gorm:"type:uuid" json:"delivery_address_id"`
	PickupBranchID   *uuid.UUID `gorm:"type:uuid" json:"pickup_branch_id"`
	DeliveryAddressLine string  `json:"delivery_address_line"`
	DeliveryApartment   string  `json:"delivery_apartment"`
	DeliveryCity        string  `json:"delivery_city"`
	DeliveryDistrict    string  `json:"delivery_district"`
	PaymentMethod     string    `json:"payment_method"`
	TransactionID     string    `json:"transaction_id"`
	BonusAmount       float64   `json:"bonus_amount"`
	Notes             string    `json:"notes"`
	Items             []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	BaseModel
	OrderID          uuid.UUID  `gorm:"type:uuid;index" json:"order_id"`
	ProductID        *uuid.UUID `gorm:"type:uuid" json:"product_id"`
	ProductVariantID *uuid.UUID `gorm:"type:uuid" json:"product_variant_id"`
	ProductName      string     `json:"product_name"`
	VariantLabel     string     `json:"variant_label"`
	Quantity         int        `json:"quantity"`
	UnitPrice        float64    `json:"unit_price"`
	LineTotal        float64    `json:"line_total"`
}

