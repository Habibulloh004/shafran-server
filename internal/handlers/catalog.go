package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/utils"
)

// CatalogHandler manages catalog related resources.
type CatalogHandler struct {
	db *gorm.DB
}

// NewCatalogHandler constructs CatalogHandler.
func NewCatalogHandler(db *gorm.DB) *CatalogHandler {
	return &CatalogHandler{db: db}
}

// ListCategories returns paginated categories.
func (h *CatalogHandler) ListCategories(c *fiber.Ctx) error {
	pg := utils.ParsePagination(c)
	var categories []models.Category
	var total int64

	if err := h.db.Model(&models.Category{}).Count(&total).Error; err != nil {
		return err
	}

	if err := h.db.Limit(pg.Limit).Offset(pg.Offset).Order("created_at desc").
		Find(&categories).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    categories,
		"pagination": fiber.Map{
			"current_page":  pg.Page,
			"items_per_page": pg.Limit,
			"total_items":   total,
		},
	})
}

// GetCategory returns a single category by ID.
func (h *CatalogHandler) GetCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var category models.Category
	if err := h.db.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "category not found")
		}
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": category})
}

// CreateCategory persists a new category.
func (h *CatalogHandler) CreateCategory(c *fiber.Ctx) error {
	var payload models.Category
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.db.Create(&payload).Error; err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": payload})
}

// UpdateCategory updates an existing category.
func (h *CatalogHandler) UpdateCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var category models.Category
	if err := h.db.First(&category, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "category not found")
		}
		return err
	}

	var payload models.Category
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	payload.ID = category.ID
	if err := h.db.Model(&category).Updates(payload).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": category})
}

// DeleteCategory removes a category by ID.
func (h *CatalogHandler) DeleteCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.db.Delete(&models.Category{}, "id = ?", id).Error; err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// CRUD for Brand, FragranceNote, Season, ProductType follow similar patterns.

func (h *CatalogHandler) ListBrands(c *fiber.Ctx) error {
	pg := utils.ParsePagination(c)
	var items []models.Brand
	var total int64

	if err := h.db.Model(&models.Brand{}).Count(&total).Error; err != nil {
		return err
	}

	if err := h.db.Preload("Category").Limit(pg.Limit).Offset(pg.Offset).
		Order("created_at desc").Find(&items).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": items, "pagination": fiber.Map{
		"current_page":  pg.Page,
		"items_per_page": pg.Limit,
		"total_items":   total,
	}})
}

func (h *CatalogHandler) GetBrand(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var item models.Brand
	if err := h.db.Preload("Category").First(&item, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "brand not found")
		}
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": item})
}

func (h *CatalogHandler) CreateBrand(c *fiber.Ctx) error {
	var payload models.Brand
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.db.Create(&payload).Error; err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": payload})
}

func (h *CatalogHandler) UpdateBrand(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var item models.Brand
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "brand not found")
		}
		return err
	}

	var payload models.Brand
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	payload.ID = item.ID
	if err := h.db.Model(&item).Updates(payload).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": item})
}

func (h *CatalogHandler) DeleteBrand(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.db.Delete(&models.Brand{}, "id = ?", id).Error; err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Generic helpers for simple lookup tables.

func (h *CatalogHandler) listSimple(c *fiber.Ctx, model interface{}) error {
	pg := utils.ParsePagination(c)
	var total int64
	if err := h.db.Model(model).Count(&total).Error; err != nil {
		return err
	}
	if err := h.db.Limit(pg.Limit).Offset(pg.Offset).Order("created_at desc").
		Find(model).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": model, "pagination": fiber.Map{
		"current_page":  pg.Page,
		"items_per_page": pg.Limit,
		"total_items":   total,
	}})
}

func (h *CatalogHandler) getSimple(c *fiber.Ctx, model interface{}) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.db.First(model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "resource not found")
		}
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": model})
}

func (h *CatalogHandler) createSimple(c *fiber.Ctx, model interface{}) error {
	if err := c.BodyParser(model); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.db.Create(model).Error; err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": model})
}

func (h *CatalogHandler) updateSimple(c *fiber.Ctx, model interface{}) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.db.First(model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "resource not found")
		}
		return err
	}
	if err := c.BodyParser(model); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.db.Save(model).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"success": true, "data": model})
}

func (h *CatalogHandler) deleteSimple(c *fiber.Ctx, model interface{}) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.db.Delete(model, "id = ?", id).Error; err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *CatalogHandler) ListFragranceNotes(c *fiber.Ctx) error {
	var notes []models.FragranceNote
	return h.listSimple(c, &notes)
}

func (h *CatalogHandler) GetFragranceNote(c *fiber.Ctx) error {
	var note models.FragranceNote
	return h.getSimple(c, &note)
}

func (h *CatalogHandler) CreateFragranceNote(c *fiber.Ctx) error {
	var note models.FragranceNote
	return h.createSimple(c, &note)
}

func (h *CatalogHandler) UpdateFragranceNote(c *fiber.Ctx) error {
	var note models.FragranceNote
	return h.updateSimple(c, &note)
}

func (h *CatalogHandler) DeleteFragranceNote(c *fiber.Ctx) error {
	var note models.FragranceNote
	return h.deleteSimple(c, &note)
}

func (h *CatalogHandler) ListSeasons(c *fiber.Ctx) error {
	var items []models.Season
	return h.listSimple(c, &items)
}

func (h *CatalogHandler) GetSeason(c *fiber.Ctx) error {
	var item models.Season
	return h.getSimple(c, &item)
}

func (h *CatalogHandler) CreateSeason(c *fiber.Ctx) error {
	var item models.Season
	return h.createSimple(c, &item)
}

func (h *CatalogHandler) UpdateSeason(c *fiber.Ctx) error {
	var item models.Season
	return h.updateSimple(c, &item)
}

func (h *CatalogHandler) DeleteSeason(c *fiber.Ctx) error {
	var item models.Season
	return h.deleteSimple(c, &item)
}

func (h *CatalogHandler) ListProductTypes(c *fiber.Ctx) error {
	var items []models.ProductType
	return h.listSimple(c, &items)
}

func (h *CatalogHandler) GetProductType(c *fiber.Ctx) error {
	var item models.ProductType
	return h.getSimple(c, &item)
}

func (h *CatalogHandler) CreateProductType(c *fiber.Ctx) error {
	var item models.ProductType
	return h.createSimple(c, &item)
}

func (h *CatalogHandler) UpdateProductType(c *fiber.Ctx) error {
	var item models.ProductType
	return h.updateSimple(c, &item)
}

func (h *CatalogHandler) DeleteProductType(c *fiber.Ctx) error {
	var item models.ProductType
	return h.deleteSimple(c, &item)
}
