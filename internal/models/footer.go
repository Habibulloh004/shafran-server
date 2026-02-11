package models

// FooterSettings stores dynamic footer content managed via admin panel.
// There should be only one row (singleton pattern).
type FooterSettings struct {
	BaseModel
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	Phone2      string `json:"phone2"`
	Email       string `json:"email"`
	WorkingHours string `json:"working_hours"`

	// Social links
	Telegram  string `json:"telegram"`
	Instagram string `json:"instagram"`
	Facebook  string `json:"facebook"`
	Youtube   string `json:"youtube"`
	Twitter   string `json:"twitter"`
	TikTok    string `json:"tiktok"`

	// Toggle social link visibility
	TelegramEnabled  bool `json:"telegram_enabled"`
	InstagramEnabled bool `json:"instagram_enabled"`
	FacebookEnabled  bool `json:"facebook_enabled"`
	YoutubeEnabled   bool `json:"youtube_enabled"`
	TwitterEnabled   bool `json:"twitter_enabled"`
	TikTokEnabled    bool `json:"tiktok_enabled"`

	// Extra text sections
	CopyrightText string `json:"copyright_text"`
}
