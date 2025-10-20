package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Product struct {
	BaseModel
	Slug              string            `gorm:"uniqueIndex" json:"slug"`
	Name              string            `json:"name"`
	ShortDescription  string            `json:"short_description"`
	LongDescription   string            `json:"long_description"`
	GenderAudience    string            `json:"gender_audience"`
	BasePrice         float64           `json:"base_price"`
	Currency          string            `json:"currency"`
	RatingAverage     float64           `json:"rating_average"`
	RatingCount       int               `json:"rating_count"`
	ReleaseYear       int               `json:"release_year"`
	Manufacturer      string            `json:"manufacturer"`
	CountryOfOrigin   string            `json:"country_of_origin"`
	IsTesterAvailable bool              `json:"is_tester_available"`
	FragranceFamily   string            `json:"fragrance_family"`
	FragranceGroup    string            `json:"fragrance_group"`
	CompositionNotes  string            `json:"composition_notes"`
	HeroImage         string            `json:"hero_image"`
	Parameters        string            `json:"parameters"`
	BrandID           *uuid.UUID        `gorm:"type:uuid" json:"brand_id"`
	Brand             *Brand            `json:"brand,omitempty"`
	CategoryID        *uuid.UUID        `gorm:"type:uuid" json:"category_id"`
	Category          *Category         `json:"category,omitempty"`
	Variants          []ProductVariant  `json:"variants,omitempty"`
	Media             []ProductMedia    `json:"media,omitempty"`
	Specifications    []ProductSpecification `json:"specifications,omitempty"`
	DescriptionBlocks []ProductDescriptionBlock `json:"description_blocks,omitempty"`
	Highlights        []ProductHighlight `json:"highlights,omitempty"`
	FragranceNotes    []FragranceNote    `gorm:"many2many:product_fragrance_notes;" json:"fragrance_notes,omitempty"`
	Seasons           []Season           `gorm:"many2many:product_seasons;" json:"seasons,omitempty"`
	ProductTypes      []ProductType      `gorm:"many2many:product_types_products;" json:"product_types,omitempty"`
	RelatedTitle      string             `json:"related_title"`
	RelatedProducts   []ProductRelation  `json:"related_products,omitempty"`
}

type ProductVariant struct {
	BaseModel
	ProductID        uuid.UUID `gorm:"type:uuid;index" json:"product_id"`
	SKU              string    `json:"sku"`
	Label            string    `json:"label"`
	VolumeML         int       `json:"volume_ml"`
	Price            float64   `json:"price"`
	Currency         string    `json:"currency"`
	IsTester         bool      `json:"is_tester"`
	InventoryQuantity int      `json:"inventory_quantity"`
	IsActive         bool      `json:"is_active"`
	InStock          bool      `json:"in_stock"`
}

type ProductMedia struct {
	BaseModel
	ProductID    uuid.UUID `gorm:"type:uuid;index" json:"product_id"`
	Type         string    `json:"type"` // gallery|marketing
	URL          string    `json:"url"`
	AltText      string    `json:"alt_text"`
	DisplayOrder int       `json:"display_order"`
}

type ProductSpecification struct {
	BaseModel
	ProductID    uuid.UUID `gorm:"type:uuid;index" json:"product_id"`
	Label        string    `json:"label"`
	Value        string    `json:"value"`
	DisplayOrder int       `json:"display_order"`
}

type ProductDescriptionBlock struct {
	BaseModel
	ProductID    uuid.UUID `gorm:"type:uuid;index" json:"product_id"`
	Content      string    `json:"content"`
	DisplayOrder int       `json:"display_order"`
}

type ProductHighlight struct {
	BaseModel
	ProductID    uuid.UUID      `gorm:"type:uuid;index" json:"product_id"`
	Type         string         `json:"type"`
	Text         string         `json:"text"`
	MediaItems   pq.StringArray `gorm:"type:text[]" json:"media_items"`
	DisplayOrder int            `json:"display_order"`
}

type ProductRelation struct {
	BaseModel
	ProductID        uuid.UUID `gorm:"type:uuid;index" json:"product_id"`
	RelatedProductID uuid.UUID `gorm:"type:uuid;index" json:"related_product_id"`
	Title            string    `json:"title"`
	DisplayOrder     int       `json:"display_order"`
}

