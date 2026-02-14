package models

type Banner struct {
	BaseModel
	Title   string `json:"title"`
	URL     string `json:"url"`
	ImageUz string `json:"image_uz"`
	ImageRu string `json:"image_ru"`
	ImageEn string `json:"image_en"`
}

type PickupBranch struct {
	BaseModel
	Name         string  `json:"name"`
	AddressLine  string  `json:"address_line"`
	District     string  `json:"district"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	WorkingHours string  `json:"working_hours"`
	ContactPhone string  `json:"contact_phone"`
	IsActive     bool    `json:"is_active"`
}

type PaymentProvider struct {
	BaseModel
	Name       string `json:"name"`
	Type       string `json:"type"`
	BrandColor string `json:"brand_color"`
	Image      string `json:"image"`
	IsActive   bool   `json:"is_active"`
}

