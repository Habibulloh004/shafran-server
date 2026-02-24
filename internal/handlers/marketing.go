package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// Allowed image MIME types for banner uploads.
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

const maxBannerFileSize = 50 * 1024 * 1024 // 50MB

// Banners

func (h *MarketingHandler) ListBanners(c *fiber.Ctx) error {
	lang := c.Query("lang")

	var items []models.Banner
	if err := h.db.Order("created_at desc").Find(&items).Error; err != nil {
		return err
	}

	// If lang is specified, return simplified response for public use
	if lang != "" {
		type PublicBanner struct {
			ID    uuid.UUID `json:"id"`
			Title string    `json:"title"`
			URL   string    `json:"url"`
			Image string    `json:"image"`
		}
		var result []PublicBanner
		for _, b := range items {
			image := getImageForLang(b, lang)
			if image == "" {
				continue
			}
			result = append(result, PublicBanner{
				ID:    b.ID,
				Title: b.Title,
				URL:   b.URL,
				Image: image,
			})
		}
		return c.JSON(fiber.Map{"success": true, "data": result})
	}

	return c.JSON(fiber.Map{"success": true, "data": items})
}

func getImageForLang(b models.Banner, lang string) string {
	switch lang {
	case "ru":
		if b.ImageRu != "" {
			return b.ImageRu
		}
	case "en":
		if b.ImageEn != "" {
			return b.ImageEn
		}
	}
	return b.ImageUz
}

func (h *MarketingHandler) CreateBanner(c *fiber.Ctx) error {
	title := c.FormValue("title")
	if strings.TrimSpace(title) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "title is required")
	}

	banner := models.Banner{
		Title: title,
		URL:   c.FormValue("url"),
	}

	// Handle file uploads for each language
	for _, lang := range []string{"uz", "ru", "en"} {
		fieldName := "image_" + lang
		file, err := c.FormFile(fieldName)
		if err != nil {
			continue // not provided
		}

		if !allowedImageTypes[file.Header.Get("Content-Type")] {
			return fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("invalid file type for %s: only jpg, png, webp allowed", fieldName))
		}

		if file.Size > maxBannerFileSize {
			return fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("file %s exceeds 50MB limit", fieldName))
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		filename := uuid.New().String() + ext
		savePath := filepath.Join("uploads", "banners", filename)

		if err := c.SaveFile(file, savePath); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to save file")
		}

		imageURL := "/uploads/banners/" + filename
		switch lang {
		case "uz":
			banner.ImageUz = imageURL
		case "ru":
			banner.ImageRu = imageURL
		case "en":
			banner.ImageEn = imageURL
		}
	}

	if err := h.db.Create(&banner).Error; err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": banner})
}

func (h *MarketingHandler) UpdateBanner(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var banner models.Banner
	if err := h.db.First(&banner, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "banner not found")
		}
		return err
	}

	// Update text fields
	if title := c.FormValue("title"); strings.TrimSpace(title) != "" {
		banner.Title = title
	}
	banner.URL = c.FormValue("url")

	// Handle file uploads — only overwrite if new file provided
	for _, lang := range []string{"uz", "ru", "en"} {
		fieldName := "image_" + lang
		file, err := c.FormFile(fieldName)
		if err != nil {
			continue
		}

		if !allowedImageTypes[file.Header.Get("Content-Type")] {
			return fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("invalid file type for %s: only jpg, png, webp allowed", fieldName))
		}

		if file.Size > maxBannerFileSize {
			return fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("file %s exceeds 50MB limit", fieldName))
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		filename := uuid.New().String() + ext
		savePath := filepath.Join("uploads", "banners", filename)

		if err := c.SaveFile(file, savePath); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to save file")
		}

		imageURL := "/uploads/banners/" + filename

		// Remove old file (best-effort)
		var oldPath string
		switch lang {
		case "uz":
			oldPath = banner.ImageUz
			banner.ImageUz = imageURL
		case "ru":
			oldPath = banner.ImageRu
			banner.ImageRu = imageURL
		case "en":
			oldPath = banner.ImageEn
			banner.ImageEn = imageURL
		}
		if oldPath != "" {
			os.Remove(strings.TrimPrefix(oldPath, "/"))
		}
	}

	if err := h.db.Save(&banner).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": banner})
}

func (h *MarketingHandler) DeleteBanner(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var banner models.Banner
	if err := h.db.First(&banner, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "banner not found")
		}
		return err
	}

	if err := h.db.Delete(&banner).Error; err != nil {
		return err
	}

	// Clean up files (best-effort)
	for _, path := range []string{banner.ImageUz, banner.ImageRu, banner.ImageEn} {
		if path != "" {
			os.Remove(strings.TrimPrefix(path, "/"))
		}
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
