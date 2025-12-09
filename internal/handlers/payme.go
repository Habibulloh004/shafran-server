package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/services"
)

// PaymeHandler manages Payme-related endpoints.
type PaymeHandler struct {
	db          *gorm.DB
	payme       *services.PaymeService
	merchantID  string
}

func NewPaymeHandler(db *gorm.DB, merchantID string) *PaymeHandler {
	return &PaymeHandler{
		db:         db,
		payme:      services.NewPaymeService(db),
		merchantID: merchantID,
	}
}

type paymeRPCRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     any             `json:"id"`
}

type paymeCheckoutRequest struct {
	OrderDetails json.RawMessage `json:"orderDetails"`
	Amount       float64         `json:"amount"`
	UserID       string          `json:"userId"`
	URL          string          `json:"url"`
}

// paymeFakeTransactionRequest is used to seed a fake Payme transaction for testing.
type paymeFakeTransactionRequest struct {
	UserID        string          `json:"userId"`
	OrderDetails  json.RawMessage `json:"orderDetails"`
	Status        int             `json:"status"`
	Amount        int64           `json:"amount"`
	OrderID       string          `json:"order_id"`
	CreateTime    int64           `json:"create_time"`
	PerformTime   int64           `json:"perform_time"`
	CancelTime    int64           `json:"cancel_time"`
	Reason        *int            `json:"reason"`
	Provider      string          `json:"provider"`
	TransactionID string          `json:"transaction_id"`
	PrepareID     string          `json:"prepare_id"`
}

// Pay handles Payme JSON-RPC style calls on /payme/pay.
func (h *PaymeHandler) Pay(c *fiber.Ctx) error {
	var req paymeRPCRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	ctx := context.Background()

	switch req.Method {
	case "CheckPerformTransaction":
		var params services.CheckPerformParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid params")
		}
		if err := h.payme.CheckPerformTransaction(ctx, params, req.ID); err != nil {
			return writePaymeError(c, err)
		}
		return c.JSON(fiber.Map{"result": fiber.Map{"allow": true}})
	case "CheckTransaction":
		var params services.CheckTransactionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid params")
		}
		result, err := h.payme.CheckTransaction(ctx, params, req.ID)
		if err != nil {
			return writePaymeError(c, err)
		}
		return c.JSON(fiber.Map{"result": result, "id": req.ID})
	case "CreateTransaction":
		var params services.CreateTransactionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid params")
		}
		result, err := h.payme.CreateTransaction(ctx, params, req.ID)
		if err != nil {
			return writePaymeError(c, err)
		}
		return c.JSON(fiber.Map{"result": result, "id": req.ID})
	case "PerformTransaction":
		var params services.PerformTransactionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid params")
		}
		result, err := h.payme.PerformTransaction(ctx, params, req.ID)
		if err != nil {
			return writePaymeError(c, err)
		}
		return c.JSON(fiber.Map{"result": result, "id": req.ID})
	case "CancelTransaction":
		var params services.CancelTransactionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid params")
		}
		result, err := h.payme.CancelTransaction(ctx, params, req.ID)
		if err != nil {
			return writePaymeError(c, err)
		}
		return c.JSON(fiber.Map{"result": result, "id": req.ID})
	case "GetStatement":
		var params services.StatementParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid params")
		}
		result, err := h.payme.GetStatement(ctx, params)
		if err != nil {
			return writePaymeError(c, err)
		}
		return c.JSON(fiber.Map{"result": fiber.Map{"transactions": result}})
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unsupported method")
	}
}

// Checkout creates a new Payme transaction and returns a checkout URL.
func (h *PaymeHandler) Checkout(c *fiber.Ctx) error {
	var req paymeCheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Amount <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid amount")
	}
	if strings.TrimSpace(req.URL) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url is required")
	}

	var userIDPtr *uuid.UUID
	if req.UserID != "" {
		if id, err := uuid.Parse(req.UserID); err == nil {
			userIDPtr = &id
		}
	}

	if userIDPtr != nil {
		if err := h.db.
			Where("user_id = ? AND status = ? AND provider = ?", *userIDPtr, services.TransactionStatePending, "payme").
			Delete(&models.PaymeTransaction{}).Error; err != nil {
			return err
		}
	}

	txn := models.PaymeTransaction{
		UserID:       userIDPtr,
		OrderDetails: req.OrderDetails,
		Status:       0,
		Provider:     "payme",
		Amount:       int64(math.Floor(req.Amount)),
	}

	if err := h.db.Create(&txn).Error; err != nil {
		return err
	}

	redirectURL := strings.TrimRight(req.URL, "/")

	if len(req.OrderDetails) > 0 {
		var details map[string]any
		if err := json.Unmarshal(req.OrderDetails, &details); err == nil {
			if v, ok := details["service_mode"]; ok {
				switch vv := v.(type) {
				case float64:
					if int(vv) != 1 {
						redirectURL = fmt.Sprintf("%s/%s", redirectURL, txn.ID.String())
					}
				case int:
					if vv != 1 {
						redirectURL = fmt.Sprintf("%s/%s", redirectURL, txn.ID.String())
					}
				}
			}
		}
	}

	amountOrder := int64(req.Amount * 100)
	payload := fmt.Sprintf("m=%s;ac.order_id=%s;a=%d;c=%s", h.merchantID, txn.ID.String(), amountOrder, redirectURL)
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	return c.JSON(fiber.Map{
		"url":      "https://checkout.payme.uz/" + encoded,
		"order_id": txn.ID,
	})
}

// CreateFakeTransaction inserts a fake Payme transaction for testing purposes.
func (h *PaymeHandler) CreateFakeTransaction(c *fiber.Ctx) error {
	var req paymeFakeTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var userIDPtr *uuid.UUID
	if req.UserID != "" {
		if id, err := uuid.Parse(req.UserID); err == nil {
			userIDPtr = &id
		}
	}

	txn := models.PaymeTransaction{
		TransactionID: req.TransactionID,
		UserID:        userIDPtr,
		OrderDetails:  req.OrderDetails,
		Status:        req.Status,
		Amount:        req.Amount,
		OrderID:       req.OrderID,
		CreateTime:    req.CreateTime,
		PerformTime:   req.PerformTime,
		CancelTime:    req.CancelTime,
		Reason:        req.Reason,
		Provider:      req.Provider,
		PrepareID:     req.PrepareID,
	}

	if txn.Provider == "" {
		txn.Provider = "payme"
	}

	if err := h.db.Create(&txn).Error; err != nil {
		return err
	}

	return c.JSON(txn)
}

func writePaymeError(c *fiber.Ctx, err error) error {
	if txErr, ok := err.(*services.TransactionError); ok {
		info := txErr.Info
		return c.JSON(fiber.Map{
			"error": fiber.Map{
				"code": info.Code,
				"message": fiber.Map{
					"uz": info.Message["uz"],
					"ru": info.Message["ru"],
					"en": info.Message["en"],
				},
				"data": txErr.Data,
			},
			"id": txErr.ID,
		})
	}
	return err
}
