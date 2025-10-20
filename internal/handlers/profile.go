package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/middleware"
	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/utils"
)

// ProfileHandler manages user profile endpoints.
type ProfileHandler struct {
	db *gorm.DB
}

// NewProfileHandler constructs ProfileHandler.
func NewProfileHandler(db *gorm.DB) *ProfileHandler {
	return &ProfileHandler{db: db}
}

// GetProfile returns authenticated user profile.
func (h *ProfileHandler) GetProfile(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":            user.ID,
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"display_name":  user.DisplayName,
			"phone":         user.Phone,
			"is_verified":   user.IsVerified,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
		},
	})
}

type updateProfileRequest struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
}

// UpdateProfile updates user profile fields.
func (h *ProfileHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var req updateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	updates := map[string]interface{}{}
	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if len(updates) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
	}
	updates["updated_at"] = time.Now()

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "message": "profile updated"})
}

// Address endpoints

// ListAddresses returns user addresses.
func (h *ProfileHandler) ListAddresses(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var addresses []models.UserAddress
	if err := h.db.Where("user_id = ?", userID).Find(&addresses).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": addresses})
}

type createAddressRequest struct {
	Label       string `json:"label"`
	AddressLine string `json:"address_line"`
	Apartment   string `json:"apartment"`
	City        string `json:"city"`
	District    string `json:"district"`
	PostalCode  string `json:"postal_code"`
	IsDefault   bool   `json:"is_default"`
}

// CreateAddress creates an address for the user.
func (h *ProfileHandler) CreateAddress(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var req createAddressRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	address := models.UserAddress{
		UserID:      userID,
		Label:       req.Label,
		AddressLine: req.AddressLine,
		Apartment:   req.Apartment,
		City:        req.City,
		District:    req.District,
		PostalCode:  req.PostalCode,
		IsDefault:   req.IsDefault,
	}

	if err := h.db.Create(&address).Error; err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{
		"id":    address.ID,
		"label": address.Label,
	}})
}

type updateAddressRequest struct {
	Label       *string `json:"label"`
	AddressLine *string `json:"address_line"`
	Apartment   *string `json:"apartment"`
	City        *string `json:"city"`
	District    *string `json:"district"`
	PostalCode  *string `json:"postal_code"`
	IsDefault   *bool   `json:"is_default"`
}

// UpdateAddress updates a user address.
func (h *ProfileHandler) UpdateAddress(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	addrID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var req updateAddressRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	updates := map[string]interface{}{}
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.AddressLine != nil {
		updates["address_line"] = *req.AddressLine
	}
	if req.Apartment != nil {
		updates["apartment"] = *req.Apartment
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.District != nil {
		updates["district"] = *req.District
	}
	if req.PostalCode != nil {
		updates["postal_code"] = *req.PostalCode
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}

	if len(updates) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
	}

	if err := h.db.Model(&models.UserAddress{}).
		Where("id = ? AND user_id = ?", addrID, userID).
		Updates(updates).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "address not found")
		}
		return err
	}

	return c.JSON(fiber.Map{"success": true, "message": "address updated"})
}

// DeleteAddress removes a user address.
func (h *ProfileHandler) DeleteAddress(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	addrID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.db.Where("id = ? AND user_id = ?", addrID, userID).
		Delete(&models.UserAddress{}).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "message": "address deleted"})
}

// ListBonusTransactions returns bonus ledger entries.
func (h *ProfileHandler) ListBonusTransactions(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	pg := utils.ParsePagination(c)
	query := h.db.Where("user_id = ?", userID).Model(&models.BonusTransaction{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}

	var items []models.BonusTransaction
	if err := query.Order("occurred_at desc").
		Limit(pg.Limit).Offset(pg.Offset).
		Find(&items).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    items,
		"pagination": fiber.Map{
			"current_page":  pg.Page,
			"items_per_page": pg.Limit,
			"total_items":   total,
		},
	})
}

