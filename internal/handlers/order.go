package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/middleware"
	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/services"
	"github.com/example/shafran/internal/utils"
)

// OrderHandler manages order endpoints.
type OrderHandler struct {
	db       *gorm.DB
	telegram *services.TelegramService
}

// NewOrderHandler constructs OrderHandler.
func NewOrderHandler(db *gorm.DB, telegram *services.TelegramService) *OrderHandler {
	return &OrderHandler{db: db, telegram: telegram}
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

	// Cash to'lov uchun Billz'ga order yaratish (async)
	// Telegram xabar Billz order yaratilgandan keyin yuboriladi
	// Payme to'lov uchun Billz PerformTransaction vaqtida yaratiladi va Telegram yuboriladi
	if req.PaymentMethod == "cash" {
		go h.dispatchBillzOrderAndNotify(order, userID, req)
	}
	// Payme uchun Telegram notification PerformTransaction da yuboriladi

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

// dispatchBillzOrderAndNotify creates a Billz order and sends Telegram notification after success
func (h *OrderHandler) dispatchBillzOrderAndNotify(order models.Order, userID uuid.UUID, req createOrderRequest) {
	log.Printf("[Order] dispatchBillzOrderAndNotify started for order %s, user %s", order.ID, userID)

	// Build Billz order payload
	billzItems := make([]services.BillzOrderItem, 0, len(req.Products))
	for _, p := range req.Products {
		qty := float64(p.Quantity)
		if qty <= 0 {
			qty = 1
		}
		log.Printf("[Order] Adding Billz item: productID=%s, qty=%v", p.ProductID, qty)
		billzItems = append(billzItems, services.BillzOrderItem{
			ProductID: p.ProductID,
			Quantity:  qty,
		})
	}

	// Try to get Billz customer ID by looking up user's phone
	var billzCustomerID string
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err == nil && user.Phone != "" {
		log.Printf("[Order] Found user phone: %s, will try to lookup Billz customer", user.Phone)
		billzCustomerID = ""
	}

	log.Printf("[Order] Creating Billz order with %d items, total: %v", len(billzItems), req.TotalAmount)
	result, err := services.CreateBillzOrderDirect(services.BillzOrderPayload{
		Items:         billzItems,
		CustomerID:    billzCustomerID,
		PaymentMethod: req.PaymentMethod,
		TotalAmount:   req.TotalAmount,
		Comment:       req.Notes,
	})

	now := time.Now()
	updates := map[string]any{
		"billz_synced_at": &now,
	}

	if err != nil {
		log.Printf("[Order] Billz order creation failed for order %s: %v", order.ID, err)
		errMsg := err.Error()
		if len(errMsg) > 1024 {
			errMsg = errMsg[:1024]
		}
		updates["billz_sync_error"] = errMsg
	} else if result != nil {
		log.Printf("[Order] Billz order %s created for order %s", result.OrderID, order.ID)
		updates["billz_order_id"] = result.OrderID
		updates["billz_order_number"] = result.OrderNumber
		updates["billz_order_type"] = result.OrderType
		updates["billz_sync_error"] = ""

		// Billz muvaffaqiyatli - Telegram xabar yuborish
		if h.telegram != nil {
			// Get user info for notification
			userName := "Не указано"
			userPhone := "Не указано"
			if user.FirstName != "" || user.LastName != "" {
				userName = user.FirstName + " " + user.LastName
			}
			if user.Phone != "" {
				userPhone = user.Phone
			}

			// Build items list
			items := make([]services.OrderItemNotification, 0, len(req.Products))
			for _, p := range req.Products {
				items = append(items, services.OrderItemNotification{
					Name:     p.ProductName,
					Quantity: p.Quantity,
					Price:    p.UnitPrice,
					Currency: req.Currency,
				})
			}

			// Use Billz order number if available
			orderNumber := result.OrderNumber
			if orderNumber == "" {
				orderNumber = order.OrderNumber
			}

			notification := services.OrderNotification{
				OrderID:       order.ID.String(),
				OrderNumber:   orderNumber,
				Items:         items,
				TotalAmount:   order.TotalAmount,
				Currency:      order.Currency,
				UserName:      userName,
				UserPhone:     userPhone,
				PaymentMethod: req.PaymentMethod,
				Status:        "pending",
			}

			if err := h.telegram.NotifyNewOrder(notification); err != nil {
				log.Printf("[Order] Telegram notification failed: %v", err)
			} else {
				log.Printf("[Order] Telegram notification sent for order %s", orderNumber)
			}
		}
	}

	if err := h.db.Model(&models.Order{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
		log.Printf("[Order] Failed to update Billz sync status for order %s: %v", order.ID, err)
	}
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


