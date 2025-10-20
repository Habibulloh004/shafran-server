package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/utils"
)

// MarketingHandler manages banners, pickup branches, payment providers.
type MarketingHandler struct {
	db *gorm.DB
}

// NewMarketingHandler constructs MarketingHandler.
func NewMarketingHandler(db *gorm.DB) *MarketingHandler {
	return &MarketingHandler{db: db}
}

// Banners

func (h *MarketingHandler) ListBanners(c *fiber.Ctx) error {
	var items []models.Banner
	if err := h.db.Find(&items).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

func (h *MarketingHandler) CreateBanner(c *fiber.Ctx) error {
	var item models.Banner
	if err := c.BodyParser(&item); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.db.Create(&item).Error; err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item})
}

func (h *MarketingHandler) UpdateBanner(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var item models.Banner
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "banner not found")
		}
		return err
	}
	if err := c.BodyParser(&item); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	item.ID = id
	if err := h.db.Save(&item).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}

func (h *MarketingHandler) DeleteBanner(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.db.Delete(&models.Banner{}, "id = ?", id).Error; err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Pickup branches

func (h *MarketingHandler) ListPickupBranches(c *fiber.Ctx) error {
	pg := utils.ParsePagination(c)
	var total int64
	if err := h.db.Model(&models.PickupBranch{}).Count(&total).Error; err != nil {
		return err
	}
	var items []models.PickupBranch
	if err := h.db.Limit(pg.Limit).Offset(pg.Offset).
		Order("created_at desc").Find(&items).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": items, "pagination": fiber.Map{
		"current_page":  pg.Page,
		"items_per_page": pg.Limit,
		"total_items":   total,
	}})
}

func (h *MarketingHandler) CreatePickupBranch(c *fiber.Ctx) error {
	var item models.PickupBranch
	if err := c.BodyParser(&item); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.db.Create(&item).Error; err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item})
}

func (h *MarketingHandler) UpdatePickupBranch(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var item models.PickupBranch
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "pickup branch not found")
		}
		return err
	}
	if err := c.BodyParser(&item); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	item.ID = id
	if err := h.db.Save(&item).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}

func (h *MarketingHandler) DeletePickupBranch(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.db.Delete(&models.PickupBranch{}, "id = ?", id).Error; err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Payment providers

func (h *MarketingHandler) ListPaymentProviders(c *fiber.Ctx) error {
	var items []models.PaymentProvider
	if err := h.db.Find(&items).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

func (h *MarketingHandler) CreatePaymentProvider(c *fiber.Ctx) error {
	var item models.PaymentProvider
	if err := c.BodyParser(&item); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.db.Create(&item).Error; err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item})
}

func (h *MarketingHandler) UpdatePaymentProvider(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var item models.PaymentProvider
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "payment provider not found")
		}
		return err
	}
	if err := c.BodyParser(&item); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	item.ID = id
	if err := h.db.Save(&item).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}

func (h *MarketingHandler) DeletePaymentProvider(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.db.Delete(&models.PaymentProvider{}, "id = ?", id).Error; err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

