package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
)

// FooterHandler manages footer settings endpoints.
type FooterHandler struct {
	db *gorm.DB
}

// NewFooterHandler constructs FooterHandler.
func NewFooterHandler(db *gorm.DB) *FooterHandler {
	return &FooterHandler{db: db}
}

// GetFooter returns the current footer settings (public endpoint).
func (h *FooterHandler) GetFooter(c *fiber.Ctx) error {
	var settings models.FooterSettings
	result := h.db.First(&settings)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Return default empty settings
			return c.JSON(fiber.Map{
				"success": true,
				"data":    models.FooterSettings{},
			})
		}
		return result.Error
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    settings,
	})
}

// UpdateFooter creates or updates footer settings (admin endpoint).
func (h *FooterHandler) UpdateFooter(c *fiber.Ctx) error {
	var input models.FooterSettings
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var existing models.FooterSettings
	result := h.db.First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new
		if err := h.db.Create(&input).Error; err != nil {
			return err
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data":    input,
		})
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing — keep the same ID
	input.ID = existing.ID
	if err := h.db.Save(&input).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    input,
	})
}
