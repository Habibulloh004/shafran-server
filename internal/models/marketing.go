package models

type Banner struct {
	BaseModel
	Title     string `json:"title"`
	ImageLight string `json:"image_light"`
	ImageDark  string `json:"image_dark"`
	URL       string `json:"url"`
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

