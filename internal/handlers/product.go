package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
	"github.com/example/shafran/internal/utils"
)

// ProductHandler manages product CRUD.
type ProductHandler struct {
	db *gorm.DB
}

// NewProductHandler constructs ProductHandler.
func NewProductHandler(db *gorm.DB) *ProductHandler {
	return &ProductHandler{db: db}
}

// ListProducts returns paginated products with optional filters.
func (h *ProductHandler) ListProducts(c *fiber.Ctx) error {
	pg := utils.ParsePagination(c)
	query := h.db.Model(&models.Product{})

	if v := c.Query("category_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			query = query.Where("category_id = ?", id)
		}
	}

	if v := c.Query("brand_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			query = query.Where("brand_id = ?", id)
		}
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		q := "%" + search + "%"
		query = query.Where("name ILIKE ? OR short_description ILIKE ?", q, q)
	}

	if minPrice := c.Query("min_price"); minPrice != "" {
		if val, err := strconv.ParseFloat(minPrice, 64); err == nil {
			query = query.Where("base_price >= ?", val)
		}
	}

	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if val, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			query = query.Where("base_price <= ?", val)
		}
	}

	if gender := c.Query("gender"); gender != "" {
		query = query.Where("gender_audience = ?", gender)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}

	var products []models.Product
	if err := query.Preload("Brand").Preload("Category").
		Limit(pg.Limit).Offset(pg.Offset).
		Order("created_at desc").
		Find(&products).Error; err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    products,
		"pagination": fiber.Map{
			"current_page":  pg.Page,
			"items_per_page": pg.Limit,
			"total_items":   total,
		},
	})
}

// GetProduct loads a product with relations.
func (h *ProductHandler) GetProduct(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var product models.Product
	if err := h.db.Preload("Brand").
		Preload("Category").
		Preload("Variants").
		Preload("Media").
		Preload("Specifications").
		Preload("DescriptionBlocks").
		Preload("Highlights").
		Preload("FragranceNotes").
		Preload("Seasons").
		Preload("ProductTypes").
		Preload("RelatedProducts").
		First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "product not found")
		}
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": product})
}

type productRequest struct {
	Slug              string               `json:"slug"`
	Name              string               `json:"name"`
	ShortDescription  string               `json:"short_description"`
	LongDescription   string               `json:"long_description"`
	GenderAudience    string               `json:"gender_audience"`
	BasePrice         float64              `json:"base_price"`
	Currency          string               `json:"currency"`
	RatingAverage     float64              `json:"rating_average"`
	RatingCount       int                  `json:"rating_count"`
	ReleaseYear       int                  `json:"release_year"`
	Manufacturer      string               `json:"manufacturer"`
	CountryOfOrigin   string               `json:"country_of_origin"`
	IsTesterAvailable bool                 `json:"is_tester_available"`
	FragranceFamily   string               `json:"fragrance_family"`
	FragranceGroup    string               `json:"fragrance_group"`
	CompositionNotes  string               `json:"composition_notes"`
	HeroImage         string               `json:"hero_image"`
	Parameters        string               `json:"parameters"`
	BrandID           string               `json:"brand_id"`
	CategoryID        string               `json:"category_id"`
	Variants          []variantRequest     `json:"variants"`
	Media             []mediaRequest       `json:"media"`
	Specifications    []specRequest        `json:"specifications"`
	DescriptionBlocks []descRequest        `json:"description_blocks"`
	Highlights        []highlightRequest   `json:"highlights"`
	FragranceNoteIDs  []string             `json:"fragrance_note_ids"`
	SeasonIDs         []string             `json:"season_ids"`
	ProductTypeIDs    []string             `json:"product_type_ids"`
	RelatedTitle      string               `json:"related_title"`
	RelatedProductIDs []string             `json:"related_product_ids"`
}

type variantRequest struct {
	ID                string  `json:"id"`
	SKU               string  `json:"sku"`
	Label             string  `json:"label"`
	VolumeML          int     `json:"volume_ml"`
	Price             float64 `json:"price"`
	Currency          string  `json:"currency"`
	IsTester          bool    `json:"is_tester"`
	InventoryQuantity int     `json:"inventory_quantity"`
	IsActive          bool    `json:"is_active"`
	InStock           *bool   `json:"in_stock"`
}

type mediaRequest struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	URL          string `json:"url"`
	AltText      string `json:"alt_text"`
	DisplayOrder int    `json:"display_order"`
}

type specRequest struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Value        string `json:"value"`
	DisplayOrder int    `json:"display_order"`
}

type descRequest struct {
	ID           string `json:"id"`
	Content      string `json:"content"`
	DisplayOrder int    `json:"display_order"`
}

type highlightRequest struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Text         string   `json:"text"`
	MediaItems   []string `json:"media_items"`
	DisplayOrder int      `json:"display_order"`
}

// CreateProduct handles product creation.
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	var req productRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	product, err := h.buildProductFromRequest(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.attachLookupRelations(tx, &product, req); err != nil {
			return err
		}
		return tx.Create(&product).Error
	}); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": product})
}

// UpdateProduct updates an existing product and replaces its associations.
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var existing models.Product
	if err := h.db.Preload("Variants").
		Preload("Media").
		Preload("Specifications").
		Preload("DescriptionBlocks").
		Preload("Highlights").
		Preload("RelatedProducts").
		First(&existing, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "product not found")
		}
		return err
	}

	var req productRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	product, err := h.buildProductFromRequest(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	product.ID = existing.ID

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.attachLookupRelations(tx, &product, req); err != nil {
			return err
		}

		product.CreatedAt = existing.CreatedAt

		// Replace dependent associations
		if err := tx.Where("product_id = ?", product.ID).Delete(&models.ProductVariant{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", product.ID).Delete(&models.ProductMedia{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", product.ID).Delete(&models.ProductSpecification{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", product.ID).Delete(&models.ProductDescriptionBlock{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", product.ID).Delete(&models.ProductHighlight{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", product.ID).Delete(&models.ProductRelation{}).Error; err != nil {
			return err
		}

		if err := tx.Model(&existing).Association("FragranceNotes").Clear(); err != nil {
			return err
		}
		if err := tx.Model(&existing).Association("Seasons").Clear(); err != nil {
			return err
		}
		if err := tx.Model(&existing).Association("ProductTypes").Clear(); err != nil {
			return err
		}

		if err := tx.Model(&existing).Omit("ID", "CreatedAt").Updates(product).Error; err != nil {
			return err
		}

		if len(product.Variants) > 0 {
			if err := tx.Create(&product.Variants).Error; err != nil {
				return err
			}
		}
		if len(product.Media) > 0 {
			if err := tx.Create(&product.Media).Error; err != nil {
				return err
			}
		}
		if len(product.Specifications) > 0 {
			if err := tx.Create(&product.Specifications).Error; err != nil {
				return err
			}
		}
		if len(product.DescriptionBlocks) > 0 {
			if err := tx.Create(&product.DescriptionBlocks).Error; err != nil {
				return err
			}
		}
		if len(product.Highlights) > 0 {
			if err := tx.Create(&product.Highlights).Error; err != nil {
				return err
			}
		}
		if len(product.RelatedProducts) > 0 {
			if err := tx.Create(&product.RelatedProducts).Error; err != nil {
				return err
			}
		}

		if len(product.FragranceNotes) > 0 {
			if err := tx.Model(&existing).Association("FragranceNotes").Replace(product.FragranceNotes); err != nil {
				return err
			}
		}
		if len(product.Seasons) > 0 {
			if err := tx.Model(&existing).Association("Seasons").Replace(product.Seasons); err != nil {
				return err
			}
		}
		if len(product.ProductTypes) > 0 {
			if err := tx.Model(&existing).Association("ProductTypes").Replace(product.ProductTypes); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true, "data": product})
}

// DeleteProduct removes a product and its associations.
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductVariant{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductMedia{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductSpecification{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductDescriptionBlock{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductHighlight{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductRelation{}).Error; err != nil {
			return err
		}

		product := models.Product{BaseModel: models.BaseModel{ID: id}}
		if err := tx.Model(&product).Association("FragranceNotes").Clear(); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Model(&product).Association("Seasons").Clear(); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Model(&product).Association("ProductTypes").Clear(); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return tx.Delete(&models.Product{}, "id = ?", id).Error
	}); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ProductHandler) buildProductFromRequest(req productRequest) (models.Product, error) {
	product := models.Product{
		Slug:              req.Slug,
		Name:              req.Name,
		ShortDescription:  req.ShortDescription,
		LongDescription:   req.LongDescription,
		GenderAudience:    req.GenderAudience,
		BasePrice:         req.BasePrice,
		Currency:          req.Currency,
		RatingAverage:     req.RatingAverage,
		RatingCount:       req.RatingCount,
		ReleaseYear:       req.ReleaseYear,
		Manufacturer:      req.Manufacturer,
		CountryOfOrigin:   req.CountryOfOrigin,
		IsTesterAvailable: req.IsTesterAvailable,
		FragranceFamily:   req.FragranceFamily,
		FragranceGroup:    req.FragranceGroup,
		CompositionNotes:  req.CompositionNotes,
		HeroImage:         req.HeroImage,
		Parameters:        req.Parameters,
		RelatedTitle:      req.RelatedTitle,
	}

	if req.BrandID != "" {
		id, err := uuid.Parse(req.BrandID)
		if err != nil {
			return product, errors.New("invalid brand_id")
		}
		product.BrandID = &id
	}
	if req.CategoryID != "" {
		id, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return product, errors.New("invalid category_id")
		}
		product.CategoryID = &id
	}

	for _, v := range req.Variants {
		variant := models.ProductVariant{
			ProductID:        product.ID,
			SKU:              v.SKU,
			Label:            v.Label,
			VolumeML:         v.VolumeML,
			Price:            v.Price,
			Currency:         v.Currency,
			IsTester:         v.IsTester,
			InventoryQuantity: v.InventoryQuantity,
			IsActive:         v.IsActive,
		}
		if v.InStock != nil {
			variant.InStock = *v.InStock
		} else {
			variant.InStock = v.InventoryQuantity > 0
		}
		product.Variants = append(product.Variants, variant)
	}

	for _, m := range req.Media {
		product.Media = append(product.Media, models.ProductMedia{
			ProductID:    product.ID,
			Type:         m.Type,
			URL:          m.URL,
			AltText:      m.AltText,
			DisplayOrder: m.DisplayOrder,
		})
	}

	for _, s := range req.Specifications {
		product.Specifications = append(product.Specifications, models.ProductSpecification{
			ProductID:    product.ID,
			Label:        s.Label,
			Value:        s.Value,
			DisplayOrder: s.DisplayOrder,
		})
	}

	for _, d := range req.DescriptionBlocks {
		product.DescriptionBlocks = append(product.DescriptionBlocks, models.ProductDescriptionBlock{
			ProductID:    product.ID,
			Content:      d.Content,
			DisplayOrder: d.DisplayOrder,
		})
	}

	for _, hlt := range req.Highlights {
		highlight := models.ProductHighlight{
			ProductID:    product.ID,
			Type:         hlt.Type,
			Text:         hlt.Text,
			MediaItems:   hlt.MediaItems,
			DisplayOrder: hlt.DisplayOrder,
		}
		product.Highlights = append(product.Highlights, highlight)
	}

	for idx, rel := range req.RelatedProductIDs {
		if rel == "" {
			continue
		}
		if relatedID, err := uuid.Parse(rel); err == nil {
			product.RelatedProducts = append(product.RelatedProducts, models.ProductRelation{
				ProductID:        product.ID,
				RelatedProductID: relatedID,
				Title:            req.RelatedTitle,
				DisplayOrder:     idx,
			})
		} else {
			return product, errors.New("invalid related_product_ids value")
		}
	}

	return product, nil
}

func (h *ProductHandler) attachLookupRelations(tx *gorm.DB, product *models.Product, req productRequest) error {
	if len(req.FragranceNoteIDs) > 0 {
		var notes []models.FragranceNote
		if err := tx.Where("id IN ?", stringSliceToUUID(req.FragranceNoteIDs)).Find(&notes).Error; err != nil {
			return err
		}
		product.FragranceNotes = notes
	}
	if len(req.SeasonIDs) > 0 {
		var seasons []models.Season
		if err := tx.Where("id IN ?", stringSliceToUUID(req.SeasonIDs)).Find(&seasons).Error; err != nil {
			return err
		}
		product.Seasons = seasons
	}
	if len(req.ProductTypeIDs) > 0 {
		var types []models.ProductType
		if err := tx.Where("id IN ?", stringSliceToUUID(req.ProductTypeIDs)).Find(&types).Error; err != nil {
			return err
		}
		product.ProductTypes = types
	}

	// Ensure product ID is set for nested entities before save
	for i := range product.Variants {
		product.Variants[i].ProductID = product.ID
	}
	for i := range product.Media {
		product.Media[i].ProductID = product.ID
	}
	for i := range product.Specifications {
		product.Specifications[i].ProductID = product.ID
	}
	for i := range product.DescriptionBlocks {
		product.DescriptionBlocks[i].ProductID = product.ID
	}
	for i := range product.Highlights {
		product.Highlights[i].ProductID = product.ID
	}
	for i := range product.RelatedProducts {
		product.RelatedProducts[i].ProductID = product.ID
	}

	return nil
}

func stringSliceToUUID(values []string) []uuid.UUID {
	var ids []uuid.UUID
	for _, value := range values {
		if value == "" {
			continue
		}
		if id, err := uuid.Parse(value); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// RegisterProductRoutes attaches product routes to fiber app.
func (h *ProductHandler) RegisterProductRoutes(router fiber.Router) {
	router.Get("/", h.ListProducts)
	router.Get("/:id", h.GetProduct)
	router.Post("/", h.CreateProduct)
	router.Put("/:id", h.UpdateProduct)
	router.Delete("/:id", h.DeleteProduct)
}
