package models

import "github.com/google/uuid"

type Category struct {
	BaseModel
	Name           string    `json:"name"`
	Slug           string    `gorm:"uniqueIndex" json:"slug"`
	GenderAudience string    `json:"gender_audience"`
	Subtitle       string    `json:"subtitle"`
	Description    string    `json:"description"`
	HeroImageLight string    `json:"hero_image_light"`
	HeroImageDark  string    `json:"hero_image_dark"`
	CardImage      string    `json:"card_image"`
	ProductCount   int       `json:"product_count"`
	Products       []Product `json:"products,omitempty"`
}

type Brand struct {
	BaseModel
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Country      string     `json:"country"`
	Image        string     `json:"image"`
	ProductCount int        `json:"product_count"`
	CategoryID   *uuid.UUID `gorm:"type:uuid" json:"category_id"`
	Category     *Category  `json:"category,omitempty"`
	Products     []Product  `json:"products,omitempty"`
}

type FragranceNote struct {
	BaseModel
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Products    []Product `gorm:"many2many:product_fragrance_notes;" json:"products,omitempty"`
}

type Season struct {
	BaseModel
	Name     string    `json:"name"`
	Products []Product `gorm:"many2many:product_seasons;" json:"products,omitempty"`
}

type ProductType struct {
	BaseModel
	Name     string    `json:"name"`
	Products []Product `gorm:"many2many:product_types_products;" json:"products,omitempty"`
}
