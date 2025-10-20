package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/middleware"
	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/utils"
)

// OrderHandler manages order endpoints.
type OrderHandler struct {
	db *gorm.DB
}

// NewOrderHandler constructs OrderHandler.
func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{db: db}
}

type orderProductRequest struct {
	ProductID        string  `json:"product_id"`
	ProductVariantID string  `json:"product_variant_id"`
	ProductName      string  `json:"product_name"`
	VariantLabel     string  `json:"variant_label"`
	Quantity         int     `json:"quantity"`
	UnitPrice        float64 `json:"unit_price"`
	LineTotal        float64 `json:"line_total"`
}

type paymentDetailsRequest struct {
	CardToken         string `json:"card_token"`
	DigitalProviderID string `json:"digital_provider_id"`
}

type createOrderRequest struct {
	DeliveryMethod     string                `json:"delivery_method"`
	DeliveryAddressID  string                `json:"delivery_address_id"`
	PickupBranchID     string                `json:"pickup_branch_id"`
	PaymentMethod      string                `json:"payment_method"`
	PaymentDetails     paymentDetailsRequest `json:"payment_details"`
	Currency           string                `json:"currency"`
	Products           []orderProductRequest `json:"products"`
	Promotion          string                `json:"promotion"`
	TotalAmount        float64               `json:"total_amount"`
	BonusAmount        float64               `json:"bonus_amount"`
	Notes              string                `json:"notes"`
}

// CreateOrder allows authenticated users to place an order.
func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var req createOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	order := models.Order{
		UserID:         userID,
		DeliveryMethod: req.DeliveryMethod,
		PaymentMethod:  req.PaymentMethod,
		Currency:       req.Currency,
		TransactionID:  req.PaymentDetails.CardToken,
		BonusAmount:    req.BonusAmount,
		Notes:          req.Notes,
		Status:         "pending",
		PlacedAt:       time.Now(),
	}

	if order.Currency == "" {
		order.Currency = "USD"
	}

	if req.DeliveryMethod == "address_delivery" && req.DeliveryAddressID != "" {
		if id, err := uuid.Parse(req.DeliveryAddressID); err == nil {
			var address models.UserAddress
			if err := h.db.First(&address, "id = ? AND user_id = ?", id, userID).Error; err == nil {
				order.DeliveryAddressID = &address.ID
				order.DeliveryAddressLine = address.AddressLine
				order.DeliveryApartment = address.Apartment
				order.DeliveryCity = address.City
				order.DeliveryDistrict = address.District
			}
		}
	}

	if req.DeliveryMethod == "store_pickup" && req.PickupBranchID != "" {
		if id, err := uuid.Parse(req.PickupBranchID); err == nil {
			order.PickupBranchID = &id
		}
	}

	var subtotal float64
	for _, p := range req.Products {
		lineTotal := p.LineTotal
		if lineTotal == 0 {
			lineTotal = p.UnitPrice * float64(p.Quantity)
		}

		item := models.OrderItem{
			ProductName:  p.ProductName,
			VariantLabel: p.VariantLabel,
			Quantity:     p.Quantity,
			UnitPrice:    p.UnitPrice,
			LineTotal:    lineTotal,
		}

		if p.ProductID != "" {
			if id, err := uuid.Parse(p.ProductID); err == nil {
				item.ProductID = &id
			}
		}
		if p.ProductVariantID != "" {
			if id, err := uuid.Parse(p.ProductVariantID); err == nil {
				item.ProductVariantID = &id
			}
		}

		subtotal += item.LineTotal
		order.Items = append(order.Items, item)
	}

	order.Subtotal = subtotal
	order.TotalAmount = req.TotalAmount
	if order.TotalAmount == 0 {
		order.TotalAmount = subtotal - order.BonusAmount
	}

	if order.OrderNumber == "" {
		order.OrderNumber = h.generateOrderNumber()
	}

	if err := h.db.Create(&order).Error; err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":           order.ID,
			"order_number": order.OrderNumber,
			"status":       order.Status,
			"placed_at":    order.PlacedAt,
			"total":        order.TotalAmount,
			"currency":     order.Currency,
		},
	})
}

// ListOrders returns orders for authenticated user.
func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	pg := utils.ParsePagination(c)
	query := h.db.Where("user_id = ?", userID).Model(&models.Order{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}

	var orders []models.Order
	if err := query.Preload("Items").
		Order("placed_at desc").
		Limit(pg.Limit).Offset(pg.Offset).
		Find(&orders).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    orders,
		"pagination": fiber.Map{
			"current_page":  pg.Page,
			"items_per_page": pg.Limit,
			"total_items":   total,
		},
	})
}

// GetOrder returns a single order for the authenticated user.
func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var order models.Order
	if err := h.db.Preload("Items").
		First(&order, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "order not found")
		}
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": order})
}

func (h *OrderHandler) generateOrderNumber() string {
	return fmt.Sprintf("#%d", time.Now().UnixNano()%1000000000)
}

